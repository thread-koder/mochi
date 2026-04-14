package workers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/thread_koder/mochi/internal/config"
	"github.com/thread_koder/mochi/internal/database"
	"github.com/thread_koder/mochi/internal/kubernetes"
	"github.com/thread_koder/mochi/internal/logger"
)

// WorkerPool owns long-running background workers and their shared shutdown context.
type WorkerPool struct {
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	resourceSync *ResourceSyncWorker
}

// NewWorkerPool builds background workers based on the provided config.
func NewWorkerPool(cfg *config.WorkerConfig) (*WorkerPool, error) {
	if cfg == nil {
		return nil, fmt.Errorf("worker config is nil")
	}

	syncInterval := time.Duration(cfg.ResourceSyncInterval) * time.Second
	retentionPeriod := time.Duration(cfg.Retention) * 24 * time.Hour
	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerPool{
		ctx:          ctx,
		cancel:       cancel,
		resourceSync: NewResourceSyncWorker(ctx, syncInterval, retentionPeriod),
	}, nil
}

// Start launches managed workers in background goroutines.
func (wp *WorkerPool) Start() {
	log := logger.WithComponent("workers")
	log.Info().Msg("Starting worker pool...")

	wp.wg.Go(func() {
		wp.resourceSync.Run()
	})

	log.Info().Msg("Worker pool started")
}

// Stop cancels the shared context and waits until all workers exit.
func (wp *WorkerPool) Stop() {
	log := logger.WithComponent("workers")
	log.Info().Msg("Stopping worker pool")

	wp.cancel()
	wp.wg.Wait()

	log.Info().Msg("Worker pool stopped")
}

// ResourceSyncWorker periodically syncs Kubernetes resources into the database.
type ResourceSyncWorker struct {
	ctx             context.Context
	interval        time.Duration
	retentionPeriod time.Duration
}

// NewResourceSyncWorker returns a worker that syncs resources and runs cleanup tasks.
func NewResourceSyncWorker(ctx context.Context, interval time.Duration, retentionPeriod time.Duration) *ResourceSyncWorker {
	return &ResourceSyncWorker{
		ctx:             ctx,
		interval:        interval,
		retentionPeriod: retentionPeriod,
	}
}

// Run executes one sync immediately, then repeats on the configured interval.
func (w *ResourceSyncWorker) Run() {
	log := logger.WithComponent("sync-worker")
	log.Info().
		Str("interval", fmt.Sprintf("%ds", int(w.interval.Seconds()))).
		Str("retention", fmt.Sprintf("%dd", int(w.retentionPeriod.Hours()/24))).
		Msg("Starting resource sync worker...")

	ticker := time.NewTicker(w.interval)
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

// sync runs one resource refresh pass and then cleanup.
func (w *ResourceSyncWorker) sync() {
	log := logger.WithComponent("sync-worker")

	// Bound each pass so slow external calls do not block the next schedule forever.
	syncCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	log.Info().Msg("Syncing resources...")
	if err := kubernetes.SyncResources(syncCtx); err != nil {
		log.Warn().Err(err).Msg("Failed to sync resources")
	} else {
		log.Info().Msg("Resources sync completed")
	}

	w.cleanup(syncCtx)
}

// cleanup deletes stale recommendations after each sync pass.
func (w *ResourceSyncWorker) cleanup(ctx context.Context) {
	log := logger.WithComponent("sync-worker")
	since := time.Now().Add(-w.retentionPeriod)

	log.Info().Msg("Cleaning up records...")
	if err := database.DeleteComputeRecommendationsOlderThan(ctx, since); err != nil {
		log.Warn().Err(err).Msg("Failed to delete old compute recommendations")
	}
	if err := database.DeleteComputeRecommendationsForDeletedWorkloads(ctx); err != nil {
		log.Warn().Err(err).Msg("Failed to delete compute recommendations for deleted workloads")
	}
	log.Info().Msg("Cleanup completed")
}
