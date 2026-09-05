package workers

import (
	"context"
	"fmt"
	"sync"

	"github.com/thread_koder/mochi/core/internal/config"
	"github.com/thread_koder/mochi/core/internal/logger"
)

// WorkerPool owns long-running background workers and their shared shutdown context.
type WorkerPool struct {
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	resourceSync *ResourceSyncWorker
	retention    *RetentionWorker
	discovery    *DependencyDiscoveryWorker
}

func NewWorkerPool(workers *config.WorkerConfig, k8s *config.KubernetesConfig) (*WorkerPool, error) {
	if workers == nil {
		return nil, fmt.Errorf("worker config is nil")
	}
	if k8s == nil {
		return nil, fmt.Errorf("kubernetes config is nil")
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerPool{
		ctx:          ctx,
		cancel:       cancel,
		resourceSync: NewResourceSyncWorker(ctx, &workers.Sync),
		retention:    NewRetentionWorker(ctx, &workers.Retention),
		discovery:    NewDependencyDiscoveryWorker(ctx, k8s.PodCIDRs, k8s.ServiceCIDRs),
	}, nil
}

func (wp *WorkerPool) Start() {
	log := logger.WithComponent("workers")
	log.Info().Msg("Starting worker pool...")

	wp.wg.Go(func() {
		wp.resourceSync.Run()
	})
	wp.wg.Go(func() {
		wp.retention.Run()
	})
	wp.wg.Go(func() {
		wp.discovery.Run()
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
