package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/distribution/distribution/v3/registry/storage/driver"

	"github.com/muchobien/lyra/internal/store"
)

func (s *Server) maintain(ctx context.Context) {
	ticker := time.NewTicker(min(time.Minute, max(s.config.JobStaleAfter/2, time.Second)))
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		if s.readOnly.Load() {
			continue
		}
		s.gate.RLock()
		s.reapStaleTasks()
		s.pruneRuns(ctx)
		s.gate.RUnlock()
	}
}

func (s *Server) reapStaleTasks() {
	stale, err := s.store.StaleTasks(time.Now().UTC().Add(-s.config.JobStaleAfter))
	if err != nil {
		slog.Error("stale tasks", "error", err)
		return
	}
	for _, id := range stale {
		task, _, err := s.store.Task(id)
		if err != nil {
			slog.Error("stale task", "task", id, "error", err)
			continue
		}
		line := store.LogLine{At: time.Now().UTC(), Content: fmt.Sprintf("lyra: the runner stopped reporting for %s; the job failed", s.config.JobStaleAfter)}
		if _, err := s.store.AppendLog(id, task.LogLength, []store.LogLine{line}); err != nil {
			slog.Error("stale task log", "task", id, "error", err)
		}
		if _, _, err := s.store.UpdateTask(id, nil, nil, "failure"); err != nil {
			slog.Error("fail stale task", "task", id, "error", err)
			continue
		}
		slog.Warn("failed a job whose runner stopped reporting", "task", id, "after", s.config.JobStaleAfter)
	}
}

func (s *Server) pruneRuns(ctx context.Context) {
	if s.config.ActionsRetention <= 0 {
		return
	}
	expired, err := s.store.ExpiredRuns(time.Now().UTC().Add(-s.config.ActionsRetention))
	if err != nil {
		slog.Error("expired runs", "error", err)
		return
	}
	for _, run := range expired {
		for _, object := range run.Objects {
			var missing driver.PathNotFoundError
			if err := s.storage.Delete(ctx, object); err != nil && !errors.As(err, &missing) {
				slog.Error("delete run log", "object", object, "error", err)
			}
		}
		if err := s.store.DeleteRun(run.ID); err != nil {
			slog.Error("delete run", "run", run.ID, "error", err)
		}
	}
	if len(expired) > 0 {
		slog.Info("deleted runs past retention", "count", len(expired), "retention", s.config.ActionsRetention)
	}
}
