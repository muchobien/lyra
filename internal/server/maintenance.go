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
		s.reapStaleTasks(ctx)
		s.pruneRuns(ctx)
		s.gate.RUnlock()
	}
}

func (s *Server) failStaleTask(ctx context.Context, id int64) error {
	task, _, err := s.store.Task(ctx, id)
	if err != nil {
		return err
	}
	line := store.LogLine{
		At:      time.Now().UTC(),
		Content: fmt.Sprintf("lyra: the runner stopped reporting for %s; the job failed", s.config.JobStaleAfter),
	}
	if _, err := s.store.AppendLog(ctx, id, task.LogLength, []store.LogLine{line}); err != nil {
		slog.Error("stale task log", "task", id, "error", err)
	}
	_, _, err = s.store.UpdateTask(ctx, id, nil, nil, statusFailure)
	return err
}

func (s *Server) reapStaleTasks(ctx context.Context) {
	stale, err := s.store.StaleTasks(ctx, time.Now().UTC().Add(-s.config.JobStaleAfter))
	if err != nil {
		slog.Error("stale tasks", "error", err)
		return
	}
	for _, id := range stale {
		if err := s.failStaleTask(ctx, id); err != nil {
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
	expired, err := s.store.ExpiredRuns(ctx, time.Now().UTC().Add(-s.config.ActionsRetention))
	if err != nil {
		slog.Error("expired runs", "error", err)
		return
	}
	for _, run := range expired {
		for _, object := range run.Objects {
			err := s.storage.Delete(ctx, object)
			if _, absent := errors.AsType[driver.PathNotFoundError](err); err != nil && !absent {
				slog.Error("delete run log", "object", object, "error", err)
			}
		}
		if err := s.store.DeleteRun(ctx, run.ID); err != nil {
			slog.Error("delete run", "run", run.ID, "error", err)
		}
	}
	if len(expired) > 0 {
		slog.Info("deleted runs past retention", "count", len(expired), "retention", s.config.ActionsRetention)
	}
}
