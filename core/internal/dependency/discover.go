package dependency

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/thread_koder/mochi/core/internal/database"
	"github.com/thread_koder/mochi/core/internal/logger"
	"github.com/thread_koder/mochi/core/internal/prometheus"
)

const (
	discoveryWindow = "1h"
	gcMaxAge        = 30 * 24 * time.Hour
)

func Discover(ctx context.Context) error {
	log := logger.WithComponent("dependency")

	series, err := FetchConnectionSeries(ctx, prometheus.QueryOptions{
		RangeDuration: discoveryWindow,
	})
	if err != nil {
		return fmt.Errorf("fetch connection series: %w", err)
	}

	now := time.Now().UTC()
	resolveOpts := DefaultResolveOptions()
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
		nodesByKey := make(map[string]*database.DependencyNode)
		for _, edge := range merged {
			addDiscoveryNode(nodesByKey, edge.From, now)
			addDiscoveryNode(nodesByKey, edge.To, now)
		}

		nodes := make([]*database.DependencyNode, 0, len(nodesByKey))
		for _, node := range nodesByKey {
			nodes = append(nodes, node)
		}

		if err := database.UpsertDependencyNodesBatch(ctx, nodes); err != nil {
			return fmt.Errorf("upsert dependency nodes: %w", err)
		}

		edges := make([]*database.DependencyEdge, 0, len(merged))
		for _, edge := range merged {
			fromNode, ok := nodesByKey[nodeKey(edge.From.Kind, edge.From.Namespace, edge.From.Name)]
			if !ok {
				return fmt.Errorf("missing node id for from %s/%s/%s", edge.From.Kind, edge.From.Namespace, edge.From.Name)
			}
			toNode, ok := nodesByKey[nodeKey(edge.To.Kind, edge.To.Namespace, edge.To.Name)]
			if !ok {
				return fmt.Errorf("missing node id for to %s/%s/%s", edge.To.Kind, edge.To.Namespace, edge.To.Name)
			}

			edges = append(edges, &database.DependencyEdge{
				FromNodeID:          fromNode.ID,
				ToNodeID:            toNode.ID,
				Protocol:            edge.Protocol,
				Port:                edge.Port,
				ViaServiceNamespace: edge.ViaServiceNamespace,
				ViaServiceName:      edge.ViaServiceName,
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

		if err := database.UpsertDependencyEdgesBatch(ctx, edges); err != nil {
			return fmt.Errorf("upsert dependency edges: %w", err)
		}

		log.Info().
			Int("series", len(series)).
			Int("edges", len(edges)).
			Int("nodes", len(nodes)).
			Msg("Discovery pass upserted dependency graph")
	}

	return pruneStale(ctx, now)
}

func pruneStale(ctx context.Context, now time.Time) error {
	if err := database.PruneStaleDependencyEdges(ctx, now.Add(-gcMaxAge)); err != nil {
		return err
	}
	if err := database.PruneOrphanDependencyNodes(ctx); err != nil {
		return err
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
	}
	if len(existing.Evidence) == 0 && len(edge.Evidence) > 0 {
		existing.Evidence = edge.Evidence
	}
	if existing.Source == "" {
		existing.Source = edge.Source
	}
}

func addDiscoveryNode(nodesByKey map[string]*database.DependencyNode, ref NodeRef, now time.Time) {
	key := nodeKey(ref.Kind, ref.Namespace, ref.Name)
	if _, ok := nodesByKey[key]; ok {
		return
	}
	nodesByKey[key] = &database.DependencyNode{
		Kind:        ref.Kind,
		Namespace:   ref.Namespace,
		Name:        ref.Name,
		FirstSeenAt: now,
		LastSeenAt:  now,
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
		fmt.Sprintf("%d", edge.Port),
	}, "\x00")
}
