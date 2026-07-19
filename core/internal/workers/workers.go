package workers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/thread_koder/mochi/core/internal/config"
	"github.com/thread_koder/mochi/core/internal/database"
	"github.com/thread_koder/mochi/core/internal/kubernetes"
	"github.com/thread_koder/mochi/core/internal/logger"
)

// WorkerPool owns long-running background workers and their shared shutdown context.
type WorkerPool struct {
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	resourceSync *ResourceSyncWorker
}

func NewWorkerPool(cfg *config.WorkerConfig) (*WorkerPool, error) {
	if cfg == nil {
		return nil, fmt.Errorf("worker config is nil")
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerPool{
		ctx:          ctx,
		cancel:       cancel,
		resourceSync: NewResourceSyncWorker(ctx, cfg),
	}, nil
}

func (wp *WorkerPool) Start() {
	log := logger.WithComponent("workers")
	log.Info().Msg("Starting worker pool...")

	wp.wg.Go(func() {
		wp.resourceSync.Run()
	})

	log.Info().Msg("Worker pool started")
}

func (wp *WorkerPool) Stop() {
	log := logger.WithComponent("workers")
	log.Info().Msg("Stopping worker pool")

	wp.cancel()
	wp.wg.Wait()

	log.Info().Msg("Worker pool stopped")
}

// ResourceSyncWorker periodically syncs Kubernetes resources into the database.
type ResourceSyncWorker struct {
	ctx context.Context
	cfg *config.WorkerConfig
}

func NewResourceSyncWorker(ctx context.Context, cfg *config.WorkerConfig) *ResourceSyncWorker {
	return &ResourceSyncWorker{
		ctx: ctx,
		cfg: cfg,
	}
}

func (w *ResourceSyncWorker) Run() {
	log := logger.WithComponent("sync-worker")
	interval := w.cfg.ResourceSyncIntervalDuration()
	retentionPeriod := w.cfg.RetentionDuration()

	log.Info().
		Str("interval", fmt.Sprintf("%ds", int(interval.Seconds()))).
		Str("retention", fmt.Sprintf("%dd", int(retentionPeriod.Hours()/24))).
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

	// Bound each pass so slow external calls do not block the next schedule forever.
	syncCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	log.Info().Msg("Starting resource sync pass...")
	kubernetes.SyncResources(syncCtx, w.cfg)
	if err := syncCtx.Err(); err != nil {
		log.Warn().Err(err).Msg("Resource sync pass ended before completion")
	} else {
		log.Info().Msg("Resource sync pass completed")
	}

	w.cleanup(syncCtx)
}

func (w *ResourceSyncWorker) cleanup(ctx context.Context) {
	log := logger.WithComponent("sync-worker")
	since := time.Now().Add(-w.cfg.RetentionDuration())

	log.Info().Msg("Starting cleanup tasks...")

	if err := database.PruneExpiredComputeRecommendations(ctx, since); err != nil {
		log.Warn().Err(err).Str("task", "compute_recommendations_retention").Msg("Cleanup task failed")
	}

	if err := ctx.Err(); err != nil {
		log.Warn().Err(err).Msg("Cleanup tasks ended before completion")
	} else {
		log.Info().Msg("Cleanup tasks completed")
	}
}
