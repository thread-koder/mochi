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

// Manages all background workers
type WorkerPool struct {
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	resourceSync *ResourceSyncWorker
	syncInterval time.Duration
}

// Creates a new worker pool
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
		syncInterval: syncInterval,
	}, nil
}

// Starts all workers
func (wp *WorkerPool) Start() {
	log := logger.WithComponent("workers")
	log.Info().Msg("Starting worker pool...")

	// Start resource sync worker
	wp.wg.Go(func() {
		wp.resourceSync.Run()
	})

	log.Info().Msg("Worker pool started")
}

// Stops all workers gracefully
func (wp *WorkerPool) Stop() {
	log := logger.WithComponent("workers")
	log.Info().Msg("Stopping worker pool")

	wp.cancel()
	wp.wg.Wait()

	log.Info().Msg("Worker pool stopped")
}

// Worker for syncing Kubernetes resources to PostgreSQL
type ResourceSyncWorker struct {
	ctx             context.Context
	interval        time.Duration
	retentionPeriod time.Duration
}

// Creates a new resource sync worker
func NewResourceSyncWorker(ctx context.Context, interval time.Duration, retentionPeriod time.Duration) *ResourceSyncWorker {
	return &ResourceSyncWorker{
		ctx:             ctx,
		interval:        interval,
		retentionPeriod: retentionPeriod,
	}
}

// Runs the resource sync worker
func (w *ResourceSyncWorker) Run() {
	log := logger.WithComponent("sync-worker")
	log.Info().
		Str("interval", fmt.Sprintf("%ds", int(w.interval.Seconds()))).
		Str("retention", fmt.Sprintf("%dd", int(w.retentionPeriod.Hours()/24))).
		Msg("Starting resources sync worker...")

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Run immediately on start
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

// Performs the actual sync operation
func (w *ResourceSyncWorker) sync() {
	log := logger.WithComponent("sync-worker")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	log.Info().Msg("Syncing resources...")
	// Sync all resources
	if err := kubernetes.SyncResources(ctx); err != nil {
		log.Warn().Err(err).Msg("Failed to sync resources")
	} else {
		log.Info().Msg("Resources sync completed")
	}

	// Run cleanup
	w.cleanup(ctx)
}

// Performs cleanup operations
func (w *ResourceSyncWorker) cleanup(ctx context.Context) {
	log := logger.WithComponent("sync-worker")
	// Calculate the since time from the retention period
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
