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
}

func NewWorkerPool(cfg *config.WorkerConfig) (*WorkerPool, error) {
	if cfg == nil {
		return nil, fmt.Errorf("worker config is nil")
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerPool{
		ctx:          ctx,
		cancel:       cancel,
		resourceSync: NewResourceSyncWorker(ctx, &cfg.Sync),
		retention:    NewRetentionWorker(ctx, &cfg.Retention),
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

	log.Info().Msg("Worker pool started")
}

func (wp *WorkerPool) Stop() {
	log := logger.WithComponent("workers")
	log.Info().Msg("Stopping worker pool")

	wp.cancel()
	wp.wg.Wait()

	log.Info().Msg("Worker pool stopped")
}
