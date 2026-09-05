package dependency

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/thread_koder/mochi/core/internal/database"
	"github.com/thread_koder/mochi/core/internal/logger"
	"github.com/thread_koder/mochi/core/internal/prometheus"
)

const discoveryWindow = "1h"

func Discover(ctx context.Context, podCIDRs, serviceCIDRs []string) error {
	log := logger.WithComponent("dependency")
	start := time.Now()

	series, err := FetchConnectionSeries(ctx, prometheus.QueryOptions{
		RangeDuration: discoveryWindow,
	})
	if err != nil {
		return fmt.Errorf("fetch connection series: %w", err)
	}

	now := time.Now().UTC()
	resolveOpts := DefaultResolveOptions(podCIDRs, serviceCIDRs)
	merged := make(map[string]*ResolvedEdge)

	for _, conn := range series {
		edge, kept, err := Resolve(ctx, conn, resolveOpts)
		if err != nil {
			log.Warn().Err(err).
				Str("src_pod_uid", conn.SrcPodUID).
				Msg("Failed to resolve connection series")
			continue
		}
		if !kept {
			continue
		}
		mergeResolvedEdge(merged, edge)
	}

	if len(merged) > 0 {
		uniqueNodes := make(map[string]struct{})
		edges := make([]*database.DependencyEdgeUpsert, 0, len(merged))
		for _, edge := range merged {
			from := database.DependencyNodeKey{
				Kind:      edge.From.Kind,
				Namespace: edge.From.Namespace,
				Name:      edge.From.Name,
			}
			to := database.DependencyNodeKey{
				Kind:      edge.To.Kind,
				Namespace: edge.To.Namespace,
				Name:      edge.To.Name,
			}
			uniqueNodes[nodeKey(from.Kind, from.Namespace, from.Name)] = struct{}{}
			uniqueNodes[nodeKey(to.Kind, to.Namespace, to.Name)] = struct{}{}
			edges = append(edges, &database.DependencyEdgeUpsert{
				From:                from,
				To:                  to,
				Protocol:            edge.Protocol,
				Port:                edge.Port,
				ViaServiceNamespace: edge.ViaServiceNamespace,
				ViaServiceName:      edge.ViaServiceName,
				ViaServicePort:      edge.ViaServicePort,
				Source:              edge.Source,
				Connects:            edge.Connects,
				TxBytes:             edge.TxBytes,
				RxBytes:             edge.RxBytes,
				ActiveConnections:   edge.ActiveConnections,
				FirstSeenAt:         now,
				LastSeenAt:          now,
				Evidence:            edge.Evidence,
			})
		}

		if err := database.UpsertDependencyGraph(ctx, edges); err != nil {
			return fmt.Errorf("upsert dependency graph: %w", err)
		}

		log.Info().
			Int("series", len(series)).
			Int("edges", len(edges)).
			Int("nodes", len(uniqueNodes)).
			Str("duration", time.Since(start).Round(time.Millisecond).String()).
			Msg("Discovery pass upserted dependency graph")
	}

	return nil
}

func mergeResolvedEdge(merged map[string]*ResolvedEdge, edge ResolvedEdge) {
	key := edgeKey(edge)
	existing, ok := merged[key]
	if !ok {
		cloned := edge
		merged[key] = &cloned
		return
	}

	existing.Connects += edge.Connects
	existing.TxBytes += edge.TxBytes
	existing.RxBytes += edge.RxBytes
	existing.ActiveConnections += edge.ActiveConnections
	if existing.ViaServiceName == nil && edge.ViaServiceName != nil {
		existing.ViaServiceNamespace = edge.ViaServiceNamespace
		existing.ViaServiceName = edge.ViaServiceName
		existing.ViaServicePort = edge.ViaServicePort
	}
	if len(existing.Evidence) == 0 && len(edge.Evidence) > 0 {
		existing.Evidence = edge.Evidence
	}
}

func nodeKey(kind, namespace, name string) string {
	return strings.Join([]string{kind, namespace, name}, "\x00")
}

func edgeKey(edge ResolvedEdge) string {
	return strings.Join([]string{
		nodeKey(edge.From.Kind, edge.From.Namespace, edge.From.Name),
		nodeKey(edge.To.Kind, edge.To.Namespace, edge.To.Name),
		edge.Protocol,
		strconv.Itoa(edge.Port),
	}, "\x00")
}
