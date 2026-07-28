package workers

import (
	"context"
	"fmt"
	"time"

	"github.com/thread_koder/mochi/core/internal/config"
	"github.com/thread_koder/mochi/core/internal/kubernetes"
	"github.com/thread_koder/mochi/core/internal/logger"
)

// ResourceSyncWorker periodically syncs Kubernetes resources into the database.
type ResourceSyncWorker struct {
	ctx context.Context
	cfg *config.WorkerSyncConfig
}

func NewResourceSyncWorker(ctx context.Context, cfg *config.WorkerSyncConfig) *ResourceSyncWorker {
	return &ResourceSyncWorker{
		ctx: ctx,
		cfg: cfg,
	}
}

func (w *ResourceSyncWorker) Run() {
	log := logger.WithComponent("sync-worker")
	interval := w.cfg.IntervalDuration()

	log.Info().
		Str("interval", fmt.Sprintf("%ds", int(interval.Seconds()))).
		Msg("Starting resource sync worker...")

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run one pass on startup so data is available before the first ticker interval.
	w.sync()

	for {
		select {
		case <-w.ctx.Done():
			log.Info().Msg("Sync worker stopped")
			return
		case <-ticker.C:
			w.sync()
		}
	}
}

func (w *ResourceSyncWorker) sync() {
	log := logger.WithComponent("sync-worker")

	ctx, cancel := context.WithTimeout(w.ctx, 10*time.Minute)
	defer cancel()

	log.Info().Msg("Starting resource sync pass...")

	kubernetes.SyncResources(ctx, w.cfg)

	if err := ctx.Err(); err != nil {
		log.Warn().Err(err).Msg("Resource sync pass ended before completion")
	} else {
		log.Info().Msg("Resource sync pass completed")
	}
}
