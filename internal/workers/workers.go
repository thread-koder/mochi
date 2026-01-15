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
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	resourceSync   *ResourceSyncWorker
	syncInterval   time.Duration
	staleThreshold time.Duration
}

// Creates a new worker pool
func NewWorkerPool(cfg *config.WorkerConfig) (*WorkerPool, error) {
	if cfg == nil {
		return nil, fmt.Errorf("worker config is nil")
	}

	syncInterval := time.Duration(cfg.ResourceSyncInterval) * time.Second
	staleThreshold := time.Duration(cfg.StaleResourceThreshold) * time.Second

	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerPool{
		ctx:            ctx,
		cancel:         cancel,
		resourceSync:   NewResourceSyncWorker(ctx, syncInterval, staleThreshold),
		syncInterval:   syncInterval,
		staleThreshold: staleThreshold,
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
		Msg("Starting resource sync worker...")

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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	log.Info().Msg("Starting resources sync process...")
	// Sync all resources
	if err := kubernetes.SyncResources(ctx); err != nil {
		log.Warn().Err(err).Msg("Failed to sync resources")
	}
	log.Info().Msg("Resources sync process completed")

	// Run cleanup after sync
	w.cleanup(ctx)
}

// Performs cleanup operations for stale resources and recommendations
func (w *ResourceSyncWorker) cleanup(ctx context.Context) {
	log := logger.WithComponent("workers")

	log.Info().Msg("Starting resource cleanup process...")
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
	if err := database.DeleteReplicaSetsNotSyncedSince(ctx, staleThreshold); err != nil {
		log.Warn().Err(err).Msg("Failed to delete stale replicasets")
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
	oldRecommendationsThreshold := time.Now().Add(-90 * 24 * time.Hour) // 90 days
	if err := database.DeleteComputeRecommendationsOlderThan(ctx, oldRecommendationsThreshold); err != nil {
		log.Warn().Err(err).Msg("Failed to delete old compute recommendations")
	}
	if err := database.CleanupComputeRecommendationsForDeletedWorkloads(ctx); err != nil {
		log.Warn().Err(err).Msg("Failed to cleanup compute recommendations for deleted workloads")
	}
	log.Info().Msg("Resource cleanup process completed")
}
