package workers

import (
	"context"
	"sync"
	"time"

	"github.com/thread_koder/mochi/internal/config"
	"github.com/thread_koder/mochi/internal/database"
	"github.com/thread_koder/mochi/internal/kubernetes"
	"github.com/thread_koder/mochi/internal/logger"
)

// Manages all background workers
type WorkerPool struct {
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	resourceSync   *ResourceSyncWorker
	syncInterval   time.Duration
	staleThreshold time.Duration
}

// Creates a new worker pool
func NewWorkerPool(cfg *config.WorkerConfig) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	syncInterval := 5 * time.Minute
	staleThreshold := 10 * time.Minute

	if cfg != nil {
		if cfg.ResourceSyncInterval > 0 {
			syncInterval = time.Duration(cfg.ResourceSyncInterval) * time.Second
		}
		if cfg.StaleResourceThreshold > 0 {
			staleThreshold = time.Duration(cfg.StaleResourceThreshold) * time.Second
		}
	}

	return &WorkerPool{
		ctx:            ctx,
		cancel:         cancel,
		resourceSync:   NewResourceSyncWorker(ctx, syncInterval, staleThreshold),
		syncInterval:   syncInterval,
		staleThreshold: staleThreshold,
	}
}

// Starts all workers
func (wp *WorkerPool) Start() {
	log := logger.WithComponent("workers")
	log.Info().Msg("Starting worker pool")

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
	ctx            context.Context
	interval       time.Duration
	staleThreshold time.Duration
}

// Creates a new resource sync worker
func NewResourceSyncWorker(ctx context.Context, interval, staleThreshold time.Duration) *ResourceSyncWorker {
	return &ResourceSyncWorker{
		ctx:            ctx,
		interval:       interval,
		staleThreshold: staleThreshold,
	}
}

// Runs the resource sync worker
func (w *ResourceSyncWorker) Run() {
	log := logger.WithComponent("workers")
	log.Info().
		Dur("interval", w.interval).
		Dur("stale_threshold", w.staleThreshold).
		Msg("Starting resource sync worker")

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Run immediately on start
	w.sync()

	for {
		select {
		case <-w.ctx.Done():
			log.Info().Msg("Resource sync worker stopped")
			return
		case <-ticker.C:
			w.sync()
		}
	}
}

// Performs the actual sync operation
func (w *ResourceSyncWorker) sync() {
	log := logger.WithComponent("workers")
	log.Info().Msg("Starting resource sync")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Sync all resources
	if err := kubernetes.SyncAllResources(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to sync resources")
		return
	}

	log.Info().Msg("Starting stale resource cleanup")
	// Clean up stale resources (resources that haven't been synced recently)
	staleThreshold := time.Now().Add(-w.staleThreshold)
	if err := database.DeletePodsNotSyncedSince(ctx, staleThreshold); err != nil {
		log.Warn().Err(err).Msg("Failed to delete stale pods")
	}
	if err := database.DeleteNodesNotSyncedSince(ctx, staleThreshold); err != nil {
		log.Warn().Err(err).Msg("Failed to delete stale nodes")
	}
	if err := database.DeleteNamespacesNotSyncedSince(ctx, staleThreshold); err != nil {
		log.Warn().Err(err).Msg("Failed to delete stale namespaces")
	}
	if err := database.DeleteDeploymentsNotSyncedSince(ctx, staleThreshold); err != nil {
		log.Warn().Err(err).Msg("Failed to delete stale deployments")
	}
	if err := database.DeleteStatefulSetsNotSyncedSince(ctx, staleThreshold); err != nil {
		log.Warn().Err(err).Msg("Failed to delete stale statefulsets")
	}
	if err := database.DeleteDaemonSetsNotSyncedSince(ctx, staleThreshold); err != nil {
		log.Warn().Err(err).Msg("Failed to delete stale daemonsets")
	}
	if err := database.DeleteServicesNotSyncedSince(ctx, staleThreshold); err != nil {
		log.Warn().Err(err).Msg("Failed to delete stale services")
	}
	if err := database.DeleteEndpointsNotSyncedSince(ctx, staleThreshold); err != nil {
		log.Warn().Err(err).Msg("Failed to delete stale endpoints")
	}
	if err := database.DeleteContainersNotSyncedSince(ctx, staleThreshold); err != nil {
		log.Warn().Err(err).Msg("Failed to delete stale containers")
	}
	log.Info().Msg("Stale resource cleanup completed")

	log.Info().Msg("Resource sync completed")
}
