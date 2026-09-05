package workers

import (
	"context"
	"fmt"
	"time"

	"github.com/thread_koder/mochi/core/internal/dependency"
	"github.com/thread_koder/mochi/core/internal/logger"
)

const discoveryInterval = 300 * time.Second

// DependencyDiscoveryWorker periodically builds the dependency graph snapshot from Prometheus.
type DependencyDiscoveryWorker struct {
	ctx          context.Context
	podCIDRs     []string
	serviceCIDRs []string
}

func NewDependencyDiscoveryWorker(ctx context.Context, podCIDRs, serviceCIDRs []string) *DependencyDiscoveryWorker {
	return &DependencyDiscoveryWorker{
		ctx:          ctx,
		podCIDRs:     podCIDRs,
		serviceCIDRs: serviceCIDRs,
	}
}

func (w *DependencyDiscoveryWorker) Run() {
	log := logger.WithComponent("discovery-worker")

	log.Info().
		Str("interval", fmt.Sprintf("%ds", int(discoveryInterval.Seconds()))).
		Msg("Starting dependency discovery worker...")

	ticker := time.NewTicker(discoveryInterval)
	defer ticker.Stop()

	w.discover()

	for {
		select {
		case <-w.ctx.Done():
			log.Info().Msg("Discovery worker stopped")
			return
		case <-ticker.C:
			w.discover()
		}
	}
}

func (w *DependencyDiscoveryWorker) discover() {
	log := logger.WithComponent("discovery-worker")

	ctx, cancel := context.WithTimeout(w.ctx, 10*time.Minute)
	defer cancel()

	log.Info().Msg("Starting dependency discovery pass...")

	if err := dependency.Discover(ctx, w.podCIDRs, w.serviceCIDRs); err != nil {
		log.Warn().Err(err).Msg("Dependency discovery pass failed")
		return
	}

	if err := ctx.Err(); err != nil {
		log.Warn().Err(err).Msg("Dependency discovery pass ended before completion")
	} else {
		log.Info().Msg("Dependency discovery pass completed")
	}
}
