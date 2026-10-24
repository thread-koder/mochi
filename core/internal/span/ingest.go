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
	parentKeys := make([]database.SpanIDPair, 0)
	seenParents := make(map[string]struct{})
	decodedSpans := make([]decodedSpan, 0, len(spans))

	for i := range spans {
		decoded, err := decodeSpan(&spans[i])
		if err != nil {
			return fmt.Errorf("span[%d]: %w", i, err)
		}
		if decoded.source == SourceMochiEBPF && len(decoded.parentSpanID) > 0 {
			key := string(decoded.traceID) + string(decoded.parentSpanID)
			if _, ok := seenParents[key]; !ok {
				seenParents[key] = struct{}{}
				parentKeys = append(parentKeys, database.SpanIDPair{
					TraceID: decoded.traceID,
					SpanID:  decoded.parentSpanID,
				})
			}
		}
		decodedSpans = append(decodedSpans, decoded)
	}

	otlpParents, err := database.HasOTLPClientParents(ctx, parentKeys)
	if err != nil {
		return fmt.Errorf("check OTLP client parents: %w", err)
	}

	rows := make([]*database.Span, 0, len(decodedSpans))
	for i := range decodedSpans {
		row, err := prepareDecoded(ctx, &decodedSpans[i], &spans[i], opts, otlpParents)
		if err != nil {
			return fmt.Errorf("span[%d]: %w", i, err)
		}
		if row == nil {
			continue
		}
		rows = append(rows, row)
	}

	if err := database.InsertSpans(ctx, rows); err != nil {
		return fmt.Errorf("insert spans: %w", err)
	}
	return nil
}

type decodedSpan struct {
	traceID      []byte
	spanID       []byte
	parentSpanID []byte
	method       string
	statusClass  string
	rpcSystem    string
	kind         string
	sampledBy    string
	source       string
	protocol     string
}

func decodeSpan(s *IngestSpan) (decodedSpan, error) {
	traceID, err := decodeID(s.TraceID, TraceIDSize)
	if err != nil {
		return decodedSpan{}, invalid("trace_id: %v", err)
	}
	spanID, err := decodeID(s.SpanID, SpanIDSize)
	if err != nil {
		return decodedSpan{}, invalid("span_id: %v", err)
	}
	var parentSpanID []byte
	if s.ParentSpanID != nil && *s.ParentSpanID != "" {
		parentSpanID, err = decodeID(*s.ParentSpanID, SpanIDSize)
		if err != nil {
			return decodedSpan{}, invalid("parent_span_id: %v", err)
		}
	}
	if s.StartAt.IsZero() || s.EndAt.IsZero() {
		return decodedSpan{}, invalid("start_at and end_at are required")
	}
	if s.EndAt.Before(s.StartAt) {
		return decodedSpan{}, invalid("end_at before start_at")
	}
	method := strings.TrimSpace(s.Method)
	if method == "" {
		return decodedSpan{}, invalid("method is required")
	}
	statusClass := strings.TrimSpace(s.StatusClass)
	if statusClass == "" {
		return decodedSpan{}, invalid("status_class is required")
	}
	rpcSystem := strings.TrimSpace(s.RPCSystem)
	switch rpcSystem {
	case RPCSystemHTTP, RPCSystemGRPC, RPCSystemDB, RPCSystemOther:
	default:
		return decodedSpan{}, invalid("rpc_system must be http, grpc, db, or other")
	}
	kind := strings.TrimSpace(s.Kind)
	switch kind {
	case KindUnspecified, KindInternal, KindServer, KindClient, KindProducer, KindConsumer:
	default:
		return decodedSpan{}, invalid("kind must be unspecified, internal, server, client, producer, or consumer")
	}
	sampledBy := strings.TrimSpace(s.SampledBy)
	switch sampledBy {
	case SampledByHead, SampledByError:
	default:
		return decodedSpan{}, invalid("sampled_by must be head or error")
	}
	source := strings.TrimSpace(s.Source)
	if source == "" {
		return decodedSpan{}, invalid("source is required")
	}
	if strings.TrimSpace(s.SrcPodUID) == "" {
		return decodedSpan{}, invalid("src_pod_uid is required")
	}
	protocol := strings.TrimSpace(s.Protocol)
	if protocol == "" {
		protocol = dependency.ProtocolTCP
	}
	return decodedSpan{
		traceID:      traceID,
		spanID:       spanID,
		parentSpanID: parentSpanID,
		method:       method,
		statusClass:  statusClass,
		rpcSystem:    rpcSystem,
		kind:         kind,
		sampledBy:    sampledBy,
		source:       source,
		protocol:     protocol,
	}, nil
}

func prepareDecoded(
	ctx context.Context,
	decoded *decodedSpan,
	ingestSpan *IngestSpan,
	opts dependency.ResolveOptions,
	otlpParents map[string]struct{},
) (*database.Span, error) {
	// Prefer app CLIENT: skip eBPF hop whose parent is already an OTLP client span.
	if decoded.source == SourceMochiEBPF && len(decoded.parentSpanID) > 0 {
		key := string(decoded.traceID) + string(decoded.parentSpanID)
		if _, ok := otlpParents[key]; ok {
			return nil, nil
		}
	}

	row := &database.Span{
		TraceID:       decoded.traceID,
		SpanID:        decoded.spanID,
		ParentSpanID:  decoded.parentSpanID,
		StartAt:       ingestSpan.StartAt.UTC(),
		EndAt:         ingestSpan.EndAt.UTC(),
		Method:        decoded.method,
		Route:         ingestSpan.Route,
		StatusClass:   decoded.statusClass,
		StatusCode:    ingestSpan.StatusCode,
		GRPCStatus:    ingestSpan.GRPCStatus,
		RPCSystem:     decoded.rpcSystem,
		Kind:          decoded.kind,
		Source:        decoded.source,
		SampledBy:     decoded.sampledBy,
		SrcPodUID:     ingestSpan.SrcPodUID,
		SrcNamespace:  ingestSpan.SrcNamespace,
		SrcPod:        ingestSpan.SrcPod,
		DstPodUID:     ingestSpan.DstPodUID,
		DstNamespace:  ingestSpan.DstNamespace,
		DstPod:        ingestSpan.DstPod,
		DstIP:         ingestSpan.DstIP,
		DstPort:       ingestSpan.DstPort,
		ActualDstIP:   ingestSpan.ActualDstIP,
		ActualDstPort: ingestSpan.ActualDstPort,
		Protocol:      decoded.protocol,
		DstHostname:   ingestSpan.DstHostname,
	}

	if err := resolveSpan(ctx, row, ingestSpan, decoded.kind, decoded.protocol, opts); err != nil {
		return nil, err
	}
	return row, nil
}

func resolveSpan(ctx context.Context, row *database.Span, s *IngestSpan, kind, protocol string, opts dependency.ResolveOptions) error {
	needEnds := (kind == KindClient || kind == KindProducer) && hasDest(s)
	if needEnds {
		cs := connectionSeries(*s)
		cs.Protocol = protocol
		from, to, kept, err := dependency.ResolveEnds(ctx, cs, opts)
		if err != nil {
			return fmt.Errorf("resolve ends: %w", err)
		}
		if kept {
			row.FromKind = new(from.Kind)
			row.FromNamespace = new(from.Namespace)
			row.FromName = new(from.Name)
			row.ToKind = new(to.Kind)
			row.ToNamespace = new(to.Namespace)
			row.ToName = new(to.Name)
		}
		return nil
	}

	from, kept, err := dependency.ResolveSource(ctx, s.SrcPodUID, opts)
	if err != nil {
		return fmt.Errorf("resolve source: %w", err)
	}
	if kept {
		row.FromKind = new(from.Kind)
		row.FromNamespace = new(from.Namespace)
		row.FromName = new(from.Name)
	}
	return nil
}
