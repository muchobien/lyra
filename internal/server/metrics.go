package server

import (
	"encoding/json"
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

var (
	reposDesc       = prometheus.NewDesc("lyra_repos", "Repositories.", nil, nil)
	runsDesc        = prometheus.NewDesc("lyra_runs", "Actions runs by status.", []string{"status"}, nil)
	jobsDesc        = prometheus.NewDesc("lyra_jobs", "Actions jobs by status.", []string{"status"}, nil)
	runnersDesc     = prometheus.NewDesc("lyra_runners_online", "Runners that polled in the last minute.", nil, nil)
	readOnlyDesc    = prometheus.NewDesc("lyra_read_only", "1 while Lyra is read-only for maintenance.", nil, nil)
	backupTimeDesc  = prometheus.NewDesc("lyra_backup_last_success_timestamp_seconds", "When the last backup finished.", nil, nil)
	backupBytesDesc = prometheus.NewDesc("lyra_backup_last_bytes", "Size of the last backup.", nil, nil)
	backupPauseDesc = prometheus.NewDesc("lyra_backup_last_pause_seconds", "How long writes paused for the last backup.", nil, nil)
	allStatuses     = []string{"waiting", "blocked", "running", "success", "failure", "cancelled", "skipped"}
)

type stateCollector struct {
	server *Server
}

func (collector stateCollector) Describe(descriptions chan<- *prometheus.Desc) {
	for _, description := range []*prometheus.Desc{reposDesc, runsDesc, jobsDesc, runnersDesc, readOnlyDesc, backupTimeDesc, backupBytesDesc, backupPauseDesc} {
		descriptions <- description
	}
}

func (collector stateCollector) Collect(metrics chan<- prometheus.Metric) {
	server := collector.server
	readOnly := 0.0
	if server.readOnly.Load() {
		readOnly = 1
	}
	metrics <- prometheus.MustNewConstMetric(readOnlyDesc, prometheus.GaugeValue, readOnly)
	counts, err := server.store.Counts()
	if err != nil {
		slog.Error("metrics", "error", err)
		return
	}
	metrics <- prometheus.MustNewConstMetric(reposDesc, prometheus.GaugeValue, float64(counts.Repos))
	for _, status := range allStatuses {
		metrics <- prometheus.MustNewConstMetric(runsDesc, prometheus.GaugeValue, float64(counts.Runs[status]), status)
		metrics <- prometheus.MustNewConstMetric(jobsDesc, prometheus.GaugeValue, float64(counts.Jobs[status]), status)
	}
	runners, err := server.store.Runners()
	if err != nil {
		slog.Error("metrics", "error", err)
		return
	}
	online := 0
	for _, runner := range runners {
		if runner.LastSeenAt != nil && time.Since(*runner.LastSeenAt) < runnerOnline {
			online++
		}
	}
	metrics <- prometheus.MustNewConstMetric(runnersDesc, prometheus.GaugeValue, float64(online))
	last, err := server.store.LatestEvent("backup.created")
	if errors.Is(err, store.ErrNotFound) {
		return
	}
	if err != nil {
		slog.Error("metrics", "error", err)
		return
	}
	var details struct {
		Bytes   float64 `json:"bytes"`
		PauseMS float64 `json:"pause_ms"`
	}
	json.Unmarshal(last.Data, &details)
	metrics <- prometheus.MustNewConstMetric(backupTimeDesc, prometheus.GaugeValue, float64(last.At.Unix()))
	metrics <- prometheus.MustNewConstMetric(backupBytesDesc, prometheus.GaugeValue, details.Bytes)
	metrics <- prometheus.MustNewConstMetric(backupPauseDesc, prometheus.GaugeValue, details.PauseMS/1000)
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
	s.requests = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "lyra_http_requests_total", Help: "HTTP requests by method and status class."}, []string{"method", "code"})
	s.durations = prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "lyra_http_request_duration_seconds", Help: "HTTP request duration by method.", Buckets: prometheus.DefBuckets}, []string{"method"})
	s.metrics.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}), stateCollector{server: s}, s.requests, s.durations)
}
