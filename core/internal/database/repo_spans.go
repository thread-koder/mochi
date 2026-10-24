package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type Span struct {
	TraceID       []byte
	SpanID        []byte
	ParentSpanID  []byte
	StartAt       time.Time
	EndAt         time.Time
	Method        string
	Route         string
	StatusClass   string
	StatusCode    int
	GRPCStatus    *int
	RPCSystem     string
	Kind          string
	Source        string
	SampledBy     string
	SrcPodUID     string
	SrcNamespace  string
	SrcPod        string
	DstPodUID     string
	DstNamespace  string
	DstPod        string
	DstIP         string
	DstPort       int
	ActualDstIP   string
	ActualDstPort int
	Protocol      string
	DstHostname   string
	FromKind      *string
	FromNamespace *string
	FromName      *string
	ToKind        *string
	ToNamespace   *string
	ToName        *string
}

const spanSelectColumns = `
	trace_id, span_id, parent_span_id, start_at, end_at,
	method, route, status_class, status_code, grpc_status,
	rpc_system, kind, source, sampled_by,
	src_pod_uid, src_namespace, src_pod,
	dst_pod_uid, dst_namespace, dst_pod,
	dst_ip, dst_port, actual_dst_ip, actual_dst_port,
	protocol, dst_hostname,
	from_kind, from_namespace, from_name,
	to_kind, to_namespace, to_name
`

type SpanIDPair struct {
	TraceID []byte
	SpanID  []byte
}

func spanPairKey(traceID, spanID []byte) string {
	return string(traceID) + string(spanID)
}

func InsertSpans(ctx context.Context, spans []*Span) error {
	if len(spans) == 0 {
		return nil
	}

	tx, err := Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	batch := &pgx.Batch{}
	for _, span := range spans {
		batch.Queue(`
			INSERT INTO spans (
				trace_id, span_id, parent_span_id, start_at, end_at,
				method, route, status_class, status_code, grpc_status,
				rpc_system, kind, source, sampled_by,
				src_pod_uid, src_namespace, src_pod,
				dst_pod_uid, dst_namespace, dst_pod,
				dst_ip, dst_port, actual_dst_ip, actual_dst_port,
				protocol, dst_hostname,
				from_kind, from_namespace, from_name,
				to_kind, to_namespace, to_name
			) VALUES (
				@trace_id, @span_id, @parent_span_id, @start_at, @end_at,
				@method, @route, @status_class, @status_code, @grpc_status,
				@rpc_system, @kind, @source, @sampled_by,
				@src_pod_uid, @src_namespace, @src_pod,
				@dst_pod_uid, @dst_namespace, @dst_pod,
				@dst_ip, @dst_port, @actual_dst_ip, @actual_dst_port,
				@protocol, @dst_hostname,
				@from_kind, @from_namespace, @from_name,
				@to_kind, @to_namespace, @to_name
			)
			ON CONFLICT (trace_id, span_id) DO NOTHING
		`, pgx.StrictNamedArgs{
			"trace_id":        span.TraceID,
			"span_id":         span.SpanID,
			"parent_span_id":  span.ParentSpanID,
			"start_at":        span.StartAt,
			"end_at":          span.EndAt,
			"method":          span.Method,
			"route":           span.Route,
			"status_class":    span.StatusClass,
			"status_code":     span.StatusCode,
			"grpc_status":     span.GRPCStatus,
			"rpc_system":      span.RPCSystem,
			"kind":            span.Kind,
			"source":          span.Source,
			"sampled_by":      span.SampledBy,
			"src_pod_uid":     span.SrcPodUID,
			"src_namespace":   span.SrcNamespace,
			"src_pod":         span.SrcPod,
			"dst_pod_uid":     span.DstPodUID,
			"dst_namespace":   span.DstNamespace,
			"dst_pod":         span.DstPod,
			"dst_ip":          span.DstIP,
			"dst_port":        span.DstPort,
			"actual_dst_ip":   span.ActualDstIP,
			"actual_dst_port": span.ActualDstPort,
			"protocol":        span.Protocol,
			"dst_hostname":    span.DstHostname,
			"from_kind":       span.FromKind,
			"from_namespace":  span.FromNamespace,
			"from_name":       span.FromName,
			"to_kind":         span.ToKind,
			"to_namespace":    span.ToNamespace,
			"to_name":         span.ToName,
		})
	}

	results := tx.SendBatch(ctx, batch)
	for range spans {
		if _, err := results.Exec(); err != nil {
			results.Close()
			return fmt.Errorf("failed to execute batch insert for spans: %w", err)
		}
	}
	if err := results.Close(); err != nil {
		return fmt.Errorf("failed to close batch results: %w", err)
	}

	otlpClients := make([]SpanIDPair, 0)
	seenClients := make(map[string]struct{})
	for _, span := range spans {
		if span.Source != "otlp" || span.Kind != "client" {
			continue
		}
		key := spanPairKey(span.TraceID, span.SpanID)
		if _, ok := seenClients[key]; ok {
			continue
		}
		seenClients[key] = struct{}{}
		otlpClients = append(otlpClients, SpanIDPair{TraceID: span.TraceID, SpanID: span.SpanID})
	}
	if len(otlpClients) > 0 {
		if err := deleteEBPFChildrenOfOTLPClients(ctx, tx, otlpClients); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func splitSpanIDPairs(pairs []SpanIDPair) (traceIDs, spanIDs [][]byte) {
	traceIDs = make([][]byte, len(pairs))
	spanIDs = make([][]byte, len(pairs))
	for index, pair := range pairs {
		traceIDs[index] = pair.TraceID
		spanIDs[index] = pair.SpanID
	}
	return traceIDs, spanIDs
}

func deleteEBPFChildrenOfOTLPClients(ctx context.Context, tx pgx.Tx, clients []SpanIDPair) error {
	traceIDs, spanIDs := splitSpanIDPairs(clients)
	_, err := tx.Exec(ctx, `
		DELETE FROM spans
		WHERE source = 'mochi-ebpf'
		  AND (trace_id, parent_span_id) IN (
			SELECT * FROM unnest(@trace_ids::bytea[], @span_ids::bytea[])
		)
	`, pgx.StrictNamedArgs{
		"trace_ids": traceIDs,
		"span_ids":  spanIDs,
	})
	if err != nil {
		return fmt.Errorf("failed to delete eBPF children of OTLP client spans: %w", err)
	}
	return nil
}

func HasOTLPClientParents(ctx context.Context, pairs []SpanIDPair) (map[string]struct{}, error) {
	found := make(map[string]struct{})
	if len(pairs) == 0 {
		return found, nil
	}
	traceIDs, spanIDs := splitSpanIDPairs(pairs)
	rows, err := Pool.Query(ctx, `
		SELECT trace_id, span_id FROM spans
		WHERE source = 'otlp'
		  AND kind = 'client'
		  AND (trace_id, span_id) IN (
			SELECT * FROM unnest(@trace_ids::bytea[], @span_ids::bytea[])
		)
	`, pgx.StrictNamedArgs{
		"trace_ids": traceIDs,
		"span_ids":  spanIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to check OTLP client parents: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var traceID, spanID []byte
		if err := rows.Scan(&traceID, &spanID); err != nil {
			return nil, fmt.Errorf("failed to scan OTLP client parent: %w", err)
		}
		found[spanPairKey(traceID, spanID)] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate OTLP client parents: %w", err)
	}
	return found, nil
}

func GetSpansByNamespace(ctx context.Context, namespace string, since time.Time, limit int) ([]*Span, error) {
	query := `
		SELECT ` + spanSelectColumns + `
		FROM spans
		WHERE (src_namespace = @namespace OR dst_namespace = @namespace)
		  AND start_at >= @since
		ORDER BY start_at DESC
		LIMIT @limit
	`
	rows, err := Pool.Query(ctx, query, pgx.StrictNamedArgs{
		"namespace": namespace,
		"since":     since,
		"limit":     limit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get spans by namespace: %w", err)
	}
	defer rows.Close()
	return collectSpans(rows)
}

func GetSpansByWorkload(ctx context.Context, kind, namespace, name string, since time.Time, limit int) ([]*Span, error) {
	query := `
		SELECT ` + spanSelectColumns + `
		FROM spans
		WHERE (
			(from_kind = @kind AND from_namespace = @namespace AND from_name = @name)
			OR (to_kind = @kind AND to_namespace = @namespace AND to_name = @name)
		)
		  AND start_at >= @since
		ORDER BY start_at DESC
		LIMIT @limit
	`
	rows, err := Pool.Query(ctx, query, pgx.StrictNamedArgs{
		"kind":      kind,
		"namespace": namespace,
		"name":      name,
		"since":     since,
		"limit":     limit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get spans by workload: %w", err)
	}
	defer rows.Close()
	return collectSpans(rows)
}

func collectSpans(rows pgx.Rows) ([]*Span, error) {
	spans := make([]*Span, 0)
	for rows.Next() {
		var s Span
		if err := rows.Scan(
			&s.TraceID, &s.SpanID, &s.ParentSpanID, &s.StartAt, &s.EndAt,
			&s.Method, &s.Route, &s.StatusClass, &s.StatusCode, &s.GRPCStatus,
			&s.RPCSystem, &s.Kind, &s.Source, &s.SampledBy,
			&s.SrcPodUID, &s.SrcNamespace, &s.SrcPod,
			&s.DstPodUID, &s.DstNamespace, &s.DstPod,
			&s.DstIP, &s.DstPort, &s.ActualDstIP, &s.ActualDstPort,
			&s.Protocol, &s.DstHostname,
			&s.FromKind, &s.FromNamespace, &s.FromName,
			&s.ToKind, &s.ToNamespace, &s.ToName,
		); err != nil {
			return nil, fmt.Errorf("failed to scan span: %w", err)
		}
		spans = append(spans, &s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate spans: %w", err)
	}
	return spans, nil
}

func PruneExpiredSpans(ctx context.Context, since time.Time) error {
	_, err := Pool.Exec(ctx,
		`DELETE FROM spans WHERE end_at < @since`,
		pgx.StrictNamedArgs{"since": since},
	)
	if err != nil {
		return fmt.Errorf("failed to prune expired spans: %w", err)
	}
	return nil
}
