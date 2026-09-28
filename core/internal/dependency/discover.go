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
	"golang.org/x/sync/errgroup"
)

const minIncreaseWindow = time.Minute

func Discover(ctx context.Context, podCIDRs, serviceCIDRs []string) error {
	log := logger.WithComponent("dependency")
	now := time.Now().UTC()
	hourStart := now.Truncate(time.Hour)
	prevHourStart := hourStart.Add(-time.Hour)
	resolveOpts := DefaultResolveOptions(podCIDRs, serviceCIDRs)

	elapsed := now.Sub(hourStart)
	currentOpts := prometheus.QueryOptions{At: now}

	var (
		currentSeries []ConnectionSeries
		err           error
	)
	if elapsed < minIncreaseWindow {
		currentSeries, err = FetchActiveConnectionSeries(ctx, currentOpts)
	} else {
		currentOpts.RangeDuration = fmt.Sprintf("%ds", int(elapsed.Seconds()))
		currentSeries, err = FetchConnectionSeries(ctx, currentOpts)
	}
	if err != nil {
		return fmt.Errorf("fetch current-hour connection series: %w", err)
	}

	alreadyFinalized, err := database.PreviousHourFinalized(ctx, prevHourStart, hourStart)
	if err != nil {
		return fmt.Errorf("check previous-hour finalization: %w", err)
	}

	currentMerged := resolveAndMerge(ctx, currentSeries, resolveOpts)

	g, gctx := errgroup.WithContext(ctx)
	var prevSeries []ConnectionSeries
	if !alreadyFinalized {
		g.Go(func() error {
			series, err := FetchConnectionSeries(gctx, prometheus.QueryOptions{
				RangeDuration: "1h",
				At:            hourStart,
			})
			prevSeries = series
			return err
		})
	}

	if err := writeCurrentHour(ctx, currentMerged, now, hourStart); err != nil {
		_ = g.Wait()
		return err
	}
	if err := g.Wait(); err != nil {
		return fmt.Errorf("fetch previous-hour connection series: %w", err)
	}
	if !alreadyFinalized {
		prevMerged := resolveAndMerge(ctx, prevSeries, resolveOpts)
		if err := writeFinalizeHour(ctx, prevMerged, prevHourStart); err != nil {
			return err
		}
	}

	log.Info().
		Int("current_series", len(currentSeries)).
		Int("current_edges", len(currentMerged)).
		Str("duration", time.Since(now).Round(time.Millisecond).String()).
		Msg("Discovery pass wrote dependency graph hours")

	return nil
}

func resolveAndMerge(
	ctx context.Context,
	series []ConnectionSeries,
	resolveOpts ResolveOptions,
) map[string]*ResolvedEdge {
	log := logger.WithComponent("dependency")
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
	return merged
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

func writeCurrentHour(
	ctx context.Context,
	merged map[string]*ResolvedEdge,
	now, hourStart time.Time,
) error {
	if len(merged) == 0 {
		return nil
	}

	edges := make([]*database.DependencyEdgeUpsert, 0, len(merged))
	hours := make([]*database.DependencyEdgeHour, 0, len(merged))
	for _, edge := range merged {
		edges = append(edges, &database.DependencyEdgeUpsert{
			From: database.DependencyNodeKey{
				Kind: edge.From.Kind, Namespace: edge.From.Namespace, Name: edge.From.Name,
			},
			To: database.DependencyNodeKey{
				Kind: edge.To.Kind, Namespace: edge.To.Namespace, Name: edge.To.Name,
			},
			Protocol:            edge.Protocol,
			Port:                edge.Port,
			ViaServiceNamespace: edge.ViaServiceNamespace,
			ViaServiceName:      edge.ViaServiceName,
			ViaServicePort:      edge.ViaServicePort,
			Source:              edge.Source,
			FirstSeenAt:         now,
			LastSeenAt:          now,
			Evidence:            edge.Evidence,
		})
		hours = append(hours, &database.DependencyEdgeHour{
			HourStart:         hourStart,
			Connects:          edge.Connects,
			TxBytes:           edge.TxBytes,
			RxBytes:           edge.RxBytes,
			ActiveConnections: edge.ActiveConnections,
		})
	}

	if err := database.UpsertDependencyEdgesWithHours(ctx, edges, hours); err != nil {
		return fmt.Errorf("upsert current-hour dependency edges and hours: %w", err)
	}
	return nil
}

func writeFinalizeHour(ctx context.Context, merged map[string]*ResolvedEdge, prevHourStart time.Time) error {
	if len(merged) == 0 {
		return nil
	}

	keys := make([]database.DependencyEdgeIdentity, 0, len(merged))
	ordered := make([]*ResolvedEdge, 0, len(merged))
	for _, edge := range merged {
		ordered = append(ordered, edge)
		keys = append(keys, database.DependencyEdgeIdentity{
			From: database.DependencyNodeKey{
				Kind: edge.From.Kind, Namespace: edge.From.Namespace, Name: edge.From.Name,
			},
			To: database.DependencyNodeKey{
				Kind: edge.To.Kind, Namespace: edge.To.Namespace, Name: edge.To.Name,
			},
			Protocol: edge.Protocol,
			Port:     edge.Port,
		})
	}

	ids, err := database.GetDependencyEdgeIDs(ctx, keys)
	if err != nil {
		return fmt.Errorf("lookup dependency edges for finalize: %w", err)
	}

	hours := make([]*database.DependencyEdgeHour, 0, len(ordered))
	for i, edge := range ordered {
		edgeID, ok := ids[keys[i]]
		if !ok {
			continue
		}
		hours = append(hours, &database.DependencyEdgeHour{
			EdgeID:            edgeID,
			HourStart:         prevHourStart,
			Connects:          edge.Connects,
			TxBytes:           edge.TxBytes,
			RxBytes:           edge.RxBytes,
			ActiveConnections: edge.ActiveConnections,
		})
	}
	if err := database.ReplaceDependencyEdgeHours(ctx, hours); err != nil {
		return fmt.Errorf("replace previous-hour edge hours: %w", err)
	}
	return nil
}
