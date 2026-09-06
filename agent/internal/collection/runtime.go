package collection

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/thread_koder/mochi/agent/internal/collection/aggregate"
	"github.com/thread_koder/mochi/agent/internal/collection/conntrack"
	"github.com/thread_koder/mochi/agent/internal/collection/dns"
	"github.com/thread_koder/mochi/agent/internal/collection/ebpf"
	"github.com/thread_koder/mochi/agent/internal/collection/identity"
	"github.com/thread_koder/mochi/agent/internal/collection/procnet"
	"github.com/thread_koder/mochi/agent/internal/config"
	"github.com/thread_koder/mochi/agent/internal/logger"
	"github.com/thread_koder/mochi/agent/internal/metrics"
)

const (
	conntrackRefreshInterval = 5 * time.Second
	procnetSeedInterval      = 30 * time.Second
)

// Runtime owns collection lifecycle (identity, conntrack, eBPF, procnet seed).
type Runtime struct {
	cancel    context.CancelFunc
	resolver  *identity.Resolver
	ctClient  *conntrack.Client
	collector *ebpf.Collector
	closeOnce sync.Once
}

func Start(cfg config.Config, registry *metrics.Registry) *Runtime {
	log := logger.WithComponent("collection")

	if !cfg.EBPFEnabled {
		log.Info().Msg("eBPF collection disabled")
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())

	store := aggregate.NewStore(registry, cfg.MaxSeries)
	listen := procnet.NewListenIndex()
	dnsCache := dns.NewCache(cfg.MaxSeries)
	resolver := identity.NewResolver(cfg.NodeName, dnsCache.DropPod)

	if err := resolver.Start(ctx); err != nil {
		cancel()
		log.Error().Err(err).Msg("Failed to start identity resolver. Continuing without collection")
		return nil
	}

	runtime := &Runtime{cancel: cancel, resolver: resolver}

	ctClient, err := conntrack.NewClient()
	if err != nil {
		log.Error().Err(err).Msg("Failed to open conntrack. Continuing without NAT enrichment")
	}
	runtime.ctClient = ctClient

	collector, err := ebpf.Load(store, resolver, ctClient, listen, dnsCache)
	if err != nil {
		log.Error().Err(err).Msg("eBPF load failed. Continuing without collection")
		runtime.Close()
		return nil
	}
	runtime.collector = collector

	if ctClient != nil {
		go ctClient.StartRefresh(ctx, conntrackRefreshInterval)
	}
	go collector.Start(ctx)
	go procnet.NewSeeder(store, resolver, ctClient, listen, dnsCache).Start(ctx, procnetSeedInterval)
	log.Info().Msg("Collection started")
	return runtime
}

func (r *Runtime) Close() {
	if r == nil {
		return
	}
	r.closeOnce.Do(func() {
		r.cancel()
		var err error
		if r.collector != nil {
			err = errors.Join(err, r.collector.Close())
		}
		if r.ctClient != nil {
			err = errors.Join(err, r.ctClient.Close())
		}
		if r.resolver != nil {
			r.resolver.Stop()
		}
		if err != nil {
			log := logger.WithComponent("collection")
			log.Error().Err(err).Msg("Failed to close collection runtime")
		}
	})
}
