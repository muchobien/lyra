package server

import (
	"context"
	"encoding/json/v2"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/muchobien/lyra/internal/store"
)

const collectTimeout = 10 * time.Second

var (
	reposDesc    = prometheus.NewDesc("lyra_repos", "Repositories.", nil, nil)
	runsDesc     = prometheus.NewDesc("lyra_runs", "Actions runs by status.", []string{"status"}, nil)
	jobsDesc     = prometheus.NewDesc("lyra_jobs", "Actions jobs by status.", []string{"status"}, nil)
	runnersDesc  = prometheus.NewDesc("lyra_runners_online", "Runners that polled in the last minute.", nil, nil)
	readOnlyDesc = prometheus.NewDesc("lyra_read_only", "1 while Lyra is read-only for maintenance.", nil, nil)

	backupTimeDesc = prometheus.NewDesc(
		"lyra_backup_last_success_timestamp_seconds",
		"When the last backup finished.",
		nil, nil,
	)
	backupBytesDesc = prometheus.NewDesc("lyra_backup_last_bytes", "Size of the last backup.", nil, nil)
	backupPauseDesc = prometheus.NewDesc(
		"lyra_backup_last_pause_seconds",
		"How long writes paused for the last backup.",
		nil, nil,
	)

	stateDescriptions = []*prometheus.Desc{
		reposDesc,
		runsDesc,
		jobsDesc,
		runnersDesc,
		readOnlyDesc,
		backupTimeDesc,
		backupBytesDesc,
		backupPauseDesc,
	}

	allStatuses = []string{
		statusWaiting,
		statusBlocked,
		statusRunning,
		statusSuccess,
		statusFailure,
		statusCancelled,
		statusSkipped,
	}
)

type stateCollector struct {
	server *Server
}

func gauge(description *prometheus.Desc, value float64, labels ...string) prometheus.Metric {
	return prometheus.MustNewConstMetric(description, prometheus.GaugeValue, value, labels...)
}

func (collector stateCollector) Describe(descriptions chan<- *prometheus.Desc) {
	for _, description := range stateDescriptions {
		descriptions <- description
	}
}

func (collector stateCollector) collectCounts(ctx context.Context, metrics chan<- prometheus.Metric) error {
	counts, err := collector.server.store.Counts(ctx)
	if err != nil {
		return err
	}
	metrics <- gauge(reposDesc, float64(counts.Repos))
	for _, status := range allStatuses {
		metrics <- gauge(runsDesc, float64(counts.Runs[status]), status)
		metrics <- gauge(jobsDesc, float64(counts.Jobs[status]), status)
	}
	return nil
}

func (collector stateCollector) collectRunners(ctx context.Context, metrics chan<- prometheus.Metric) error {
	runners, err := collector.server.store.Runners(ctx)
	if err != nil {
		return err
	}
	online := 0
	for _, runner := range runners {
		if runner.LastSeenAt != nil && time.Since(*runner.LastSeenAt) < runnerOnline {
			online++
		}
	}
	metrics <- gauge(runnersDesc, float64(online))
	return nil
}

func (collector stateCollector) collectBackup(ctx context.Context, metrics chan<- prometheus.Metric) error {
	last, err := collector.server.store.LatestEvent(ctx, "backup.created")
	if errors.Is(err, store.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	var details BackupCreated
	if err := json.Unmarshal(last.Data, &details); err != nil {
		return err
	}
	metrics <- gauge(backupTimeDesc, float64(last.At.Unix()))
	metrics <- gauge(backupBytesDesc, float64(details.Bytes))
	metrics <- gauge(backupPauseDesc, float64(details.PauseMS)/1000)
	return nil
}

func (collector stateCollector) Collect(metrics chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), collectTimeout)
	defer cancel()
	readOnly := 0.0
	if collector.server.readOnly.Load() {
		readOnly = 1
	}
	metrics <- gauge(readOnlyDesc, readOnly)
	steps := []func(context.Context, chan<- prometheus.Metric) error{
		collector.collectCounts,
		collector.collectRunners,
		collector.collectBackup,
	}
	for _, step := range steps {
		if err := step(ctx, metrics); err != nil {
			slog.Error("metrics", "error", err)
			return
		}
	}
}

func (s *Server) instrument(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(recorder, r)
		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}
		s.requests.WithLabelValues(r.Method, strconv.Itoa(status/100)+"xx").Inc()
		s.durations.WithLabelValues(r.Method).Observe(time.Since(start).Seconds())
	})
}

func (s *Server) MetricsHandler() http.Handler {
	return promhttp.HandlerFor(s.metrics, promhttp.HandlerOpts{})
}

func (s *Server) registerMetrics() {
	s.metrics = prometheus.NewRegistry()
	s.requests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "lyra_http_requests_total",
			Help: "HTTP requests by method and status class.",
		},
		[]string{"method", "code"},
	)
	s.durations = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "lyra_http_request_duration_seconds",
			Help:    "HTTP request duration by method.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method"},
	)
	s.metrics.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		stateCollector{server: s},
		s.requests,
		s.durations,
	)
}
