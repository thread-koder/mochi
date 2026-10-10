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
	rpc_system, source, sampled_by,
	src_pod_uid, src_namespace, src_pod,
	dst_pod_uid, dst_namespace, dst_pod,
	dst_ip, dst_port, actual_dst_ip, actual_dst_port,
	protocol, dst_hostname,
	from_kind, from_namespace, from_name,
	to_kind, to_namespace, to_name
`

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
	for _, s := range spans {
		batch.Queue(`
			INSERT INTO spans (
				trace_id, span_id, parent_span_id, start_at, end_at,
				method, route, status_class, status_code, grpc_status,
				rpc_system, source, sampled_by,
				src_pod_uid, src_namespace, src_pod,
				dst_pod_uid, dst_namespace, dst_pod,
				dst_ip, dst_port, actual_dst_ip, actual_dst_port,
				protocol, dst_hostname,
				from_kind, from_namespace, from_name,
				to_kind, to_namespace, to_name
			) VALUES (
				@trace_id, @span_id, @parent_span_id, @start_at, @end_at,
				@method, @route, @status_class, @status_code, @grpc_status,
				@rpc_system, @source, @sampled_by,
				@src_pod_uid, @src_namespace, @src_pod,
				@dst_pod_uid, @dst_namespace, @dst_pod,
				@dst_ip, @dst_port, @actual_dst_ip, @actual_dst_port,
				@protocol, @dst_hostname,
				@from_kind, @from_namespace, @from_name,
				@to_kind, @to_namespace, @to_name
			)
			ON CONFLICT (trace_id, span_id) DO NOTHING
		`, pgx.StrictNamedArgs{
			"trace_id":        s.TraceID,
			"span_id":         s.SpanID,
			"parent_span_id":  s.ParentSpanID,
			"start_at":        s.StartAt,
			"end_at":          s.EndAt,
			"method":          s.Method,
			"route":           s.Route,
			"status_class":    s.StatusClass,
			"status_code":     s.StatusCode,
			"grpc_status":     s.GRPCStatus,
			"rpc_system":      s.RPCSystem,
			"source":          s.Source,
			"sampled_by":      s.SampledBy,
			"src_pod_uid":     s.SrcPodUID,
			"src_namespace":   s.SrcNamespace,
			"src_pod":         s.SrcPod,
			"dst_pod_uid":     s.DstPodUID,
			"dst_namespace":   s.DstNamespace,
			"dst_pod":         s.DstPod,
			"dst_ip":          s.DstIP,
			"dst_port":        s.DstPort,
			"actual_dst_ip":   s.ActualDstIP,
			"actual_dst_port": s.ActualDstPort,
			"protocol":        s.Protocol,
			"dst_hostname":    s.DstHostname,
			"from_kind":       s.FromKind,
			"from_namespace":  s.FromNamespace,
			"from_name":       s.FromName,
			"to_kind":         s.ToKind,
			"to_namespace":    s.ToNamespace,
			"to_name":         s.ToName,
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
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
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
			&s.RPCSystem, &s.Source, &s.SampledBy,
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
