package span

import (
	"context"
	"fmt"
	"strings"

	"github.com/thread_koder/mochi/core/internal/database"
	"github.com/thread_koder/mochi/core/internal/dependency"
)

type InvalidError struct {
	msg string
}

func (e *InvalidError) Error() string { return e.msg }

func invalid(format string, args ...any) error {
	return &InvalidError{msg: fmt.Sprintf(format, args...)}
}

// Unresolved hops are still inserted with null from/to.
func Ingest(ctx context.Context, spans []IngestSpan) error {
	if len(spans) == 0 {
		return nil
	}

	opts := dependency.DefaultResolveOptions(nil, nil)
	rows := make([]*database.Span, 0, len(spans))
	for i := range spans {
		row, err := prepareSpan(ctx, &spans[i], opts)
		if err != nil {
			return fmt.Errorf("span[%d]: %w", i, err)
		}
		rows = append(rows, row)
	}

	if err := database.InsertSpans(ctx, rows); err != nil {
		return fmt.Errorf("insert spans: %w", err)
	}
	return nil
}

func prepareSpan(ctx context.Context, s *IngestSpan, opts dependency.ResolveOptions) (*database.Span, error) {
	traceID, err := decodeID(s.TraceID, 16)
	if err != nil {
		return nil, invalid("trace_id: %v", err)
	}
	spanID, err := decodeID(s.SpanID, 8)
	if err != nil {
		return nil, invalid("span_id: %v", err)
	}
	if s.StartAt.IsZero() || s.EndAt.IsZero() {
		return nil, invalid("start_at and end_at are required")
	}
	if s.EndAt.Before(s.StartAt) {
		return nil, invalid("end_at before start_at")
	}
	method := strings.TrimSpace(s.Method)
	if method == "" {
		return nil, invalid("method is required")
	}
	statusClass := strings.TrimSpace(s.StatusClass)
	if statusClass == "" {
		return nil, invalid("status_class is required")
	}
	rpcSystem := strings.TrimSpace(s.RPCSystem)
	switch rpcSystem {
	case RPCSystemHTTP, RPCSystemGRPC:
	default:
		return nil, invalid("rpc_system must be http or grpc")
	}
	sampledBy := strings.TrimSpace(s.SampledBy)
	switch sampledBy {
	case SampledByHead, SampledByError:
	default:
		return nil, invalid("sampled_by must be head or error")
	}
	source := strings.TrimSpace(s.Source)
	if source == "" {
		source = SourceMochiEBPF
	}
	if strings.TrimSpace(s.SrcPodUID) == "" {
		return nil, invalid("src_pod_uid is required")
	}
	protocol := strings.TrimSpace(s.Protocol)
	if protocol == "" {
		protocol = dependency.ProtocolTCP
	}

	row := &database.Span{
		TraceID:       traceID,
		SpanID:        spanID,
		StartAt:       s.StartAt.UTC(),
		EndAt:         s.EndAt.UTC(),
		Method:        method,
		Route:         s.Route,
		StatusClass:   statusClass,
		StatusCode:    s.StatusCode,
		GRPCStatus:    s.GRPCStatus,
		RPCSystem:     rpcSystem,
		Source:        source,
		SampledBy:     sampledBy,
		SrcPodUID:     s.SrcPodUID,
		SrcNamespace:  s.SrcNamespace,
		SrcPod:        s.SrcPod,
		DstPodUID:     s.DstPodUID,
		DstNamespace:  s.DstNamespace,
		DstPod:        s.DstPod,
		DstIP:         s.DstIP,
		DstPort:       s.DstPort,
		ActualDstIP:   s.ActualDstIP,
		ActualDstPort: s.ActualDstPort,
		Protocol:      protocol,
		DstHostname:   s.DstHostname,
	}

	cs := connectionSeries(*s)
	cs.Protocol = protocol
	from, to, kept, err := dependency.ResolveEnds(ctx, cs, opts)
	if err != nil {
		return nil, fmt.Errorf("resolve ends: %w", err)
	}
	if kept {
		row.FromKind = new(from.Kind)
		row.FromNamespace = new(from.Namespace)
		row.FromName = new(from.Name)
		row.ToKind = new(to.Kind)
		row.ToNamespace = new(to.Namespace)
		row.ToName = new(to.Name)
	}
	return row, nil
}
