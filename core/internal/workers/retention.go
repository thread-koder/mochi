package workers

import (
	"context"
	"fmt"
	"time"

	"github.com/thread_koder/mochi/core/internal/config"
	"github.com/thread_koder/mochi/core/internal/database"
	"github.com/thread_koder/mochi/core/internal/logger"
)

type retentionTask struct {
	name string
	run  func(ctx context.Context, since time.Time) error
}

// RetentionWorker periodically deletes aged derived records.
type RetentionWorker struct {
	ctx context.Context
	cfg *config.WorkerRetentionConfig
}

func NewRetentionWorker(ctx context.Context, cfg *config.WorkerRetentionConfig) *RetentionWorker {
	return &RetentionWorker{
		ctx: ctx,
		cfg: cfg,
	}
}

func (w *RetentionWorker) Run() {
	log := logger.WithComponent("retention-worker")
	interval := w.cfg.IntervalDuration()
	maxAge := w.cfg.MaxAgeDuration()

	log.Info().
		Str("interval", fmt.Sprintf("%ds", int(interval.Seconds()))).
		Str("max_age", fmt.Sprintf("%dd", int(maxAge.Hours()/24))).
		Msg("Starting retention worker...")

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	w.run()

	for {
		select {
		case <-w.ctx.Done():
			log.Info().Msg("Retention worker stopped")
			return
		case <-ticker.C:
			w.run()
		}
	}
}

func (w *RetentionWorker) run() {
	log := logger.WithComponent("retention-worker")

	ctx, cancel := context.WithTimeout(w.ctx, 5*time.Minute)
	defer cancel()

	since := time.Now().Add(-w.cfg.MaxAgeDuration())
	tasks := []retentionTask{
		{name: "compute_recommendations", run: database.PruneExpiredComputeRecommendations},
		{name: "pod_attributions", run: database.PruneExpiredPodAttributions},
	}

	log.Info().Msg("Starting retention pass...")

	for _, task := range tasks {
		if err := task.run(ctx, since); err != nil {
			log.Warn().Err(err).Str("task", task.name).Msg("Retention task failed")
		}
	}

	if err := ctx.Err(); err != nil {
		log.Warn().Err(err).Msg("Retention pass ended before completion")
	} else {
		log.Info().Msg("Retention pass completed")
	}
}
