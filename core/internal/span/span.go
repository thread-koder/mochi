package span

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/thread_koder/mochi/core/internal/database"
	"github.com/thread_koder/mochi/core/internal/dependency"
)

const (
	SourceMochiEBPF = "mochi-ebpf"

	RPCSystemHTTP = "http"
	RPCSystemGRPC = "grpc"

	SampledByHead  = "head"
	SampledByError = "error"

	DefaultLimit = 100
	MaxLimit     = 1000
)

type IngestSpan struct {
	TraceID       string    `json:"trace_id"`
	SpanID        string    `json:"span_id"`
	StartAt       time.Time `json:"start_at"`
	EndAt         time.Time `json:"end_at"`
	Method        string    `json:"method"`
	Route         string    `json:"route"`
	StatusClass   string    `json:"status_class"`
	StatusCode    int       `json:"status_code"`
	GRPCStatus    *int      `json:"grpc_status"`
	RPCSystem     string    `json:"rpc_system"`
	Source        string    `json:"source"`
	SampledBy     string    `json:"sampled_by"`
	SrcPodUID     string    `json:"src_pod_uid"`
	SrcNamespace  string    `json:"src_namespace"`
	SrcPod        string    `json:"src_pod"`
	DstPodUID     string    `json:"dst_pod_uid"`
	DstNamespace  string    `json:"dst_namespace"`
	DstPod        string    `json:"dst_pod"`
	DstIP         string    `json:"dst_ip"`
	DstPort       int       `json:"dst_port"`
	ActualDstIP   string    `json:"actual_dst_ip"`
	ActualDstPort int       `json:"actual_dst_port"`
	Protocol      string    `json:"protocol"`
	DstHostname   string    `json:"dst_hostname"`
}

type IngestRequest struct {
	Spans []IngestSpan `json:"spans"`
}

type WorkloadRef struct {
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

type SpanDTO struct {
	TraceID         string       `json:"trace_id"`
	SpanID          string       `json:"span_id"`
	StartAt         time.Time    `json:"start_at"`
	EndAt           time.Time    `json:"end_at"`
	DurationSeconds float64      `json:"duration_seconds"`
	Method          string       `json:"method"`
	Route           string       `json:"route"`
	StatusClass     string       `json:"status_class"`
	StatusCode      int          `json:"status_code"`
	GRPCStatus      *int         `json:"grpc_status"`
	RPCSystem       string       `json:"rpc_system"`
	Source          string       `json:"source"`
	SampledBy       string       `json:"sampled_by"`
	SrcPodUID       string       `json:"src_pod_uid"`
	SrcNamespace    string       `json:"src_namespace"`
	SrcPod          string       `json:"src_pod"`
	DstPodUID       string       `json:"dst_pod_uid"`
	DstNamespace    string       `json:"dst_namespace"`
	DstPod          string       `json:"dst_pod"`
	DstIP           string       `json:"dst_ip"`
	DstPort         int          `json:"dst_port"`
	ActualDstIP     string       `json:"actual_dst_ip"`
	ActualDstPort   int          `json:"actual_dst_port"`
	Protocol        string       `json:"protocol"`
	DstHostname     string       `json:"dst_hostname"`
	From            *WorkloadRef `json:"from"`
	To              *WorkloadRef `json:"to"`
}

type SpansResponse struct {
	Spans []SpanDTO `json:"spans"`
}

func clampLimit(limit int) int {
	if limit <= 0 {
		return DefaultLimit
	}
	return min(limit, MaxLimit)
}

func toDTO(s *database.Span) SpanDTO {
	return SpanDTO{
		TraceID:         hex.EncodeToString(s.TraceID),
		SpanID:          hex.EncodeToString(s.SpanID),
		StartAt:         s.StartAt,
		EndAt:           s.EndAt,
		DurationSeconds: s.EndAt.Sub(s.StartAt).Seconds(),
		Method:          s.Method,
		Route:           s.Route,
		StatusClass:     s.StatusClass,
		StatusCode:      s.StatusCode,
		GRPCStatus:      s.GRPCStatus,
		RPCSystem:       s.RPCSystem,
		Source:          s.Source,
		SampledBy:       s.SampledBy,
		SrcPodUID:       s.SrcPodUID,
		SrcNamespace:    s.SrcNamespace,
		SrcPod:          s.SrcPod,
		DstPodUID:       s.DstPodUID,
		DstNamespace:    s.DstNamespace,
		DstPod:          s.DstPod,
		DstIP:           s.DstIP,
		DstPort:         s.DstPort,
		ActualDstIP:     s.ActualDstIP,
		ActualDstPort:   s.ActualDstPort,
		Protocol:        s.Protocol,
		DstHostname:     s.DstHostname,
		From:            workloadRef(s.FromKind, s.FromNamespace, s.FromName),
		To:              workloadRef(s.ToKind, s.ToNamespace, s.ToName),
	}
}

func workloadRef(kind, namespace, name *string) *WorkloadRef {
	if kind == nil || namespace == nil || name == nil {
		return nil
	}
	if *kind == "" || *name == "" {
		return nil
	}
	return &WorkloadRef{Kind: *kind, Namespace: *namespace, Name: *name}
}

func decodeID(hexID string, want int) ([]byte, error) {
	raw, err := hex.DecodeString(hexID)
	if err != nil {
		return nil, fmt.Errorf("invalid hex id: %w", err)
	}
	if len(raw) != want {
		return nil, fmt.Errorf("id length %d, want %d", len(raw), want)
	}
	var zero [16]byte
	if bytes.Equal(raw, zero[:want]) {
		return nil, fmt.Errorf("id is all-zero")
	}
	return raw, nil
}

// connectionSeries adapts an ingest hop for ResolveEnds.
// Connects=1 passes the volume gate. Protocol must be tcp|udp.
func connectionSeries(s IngestSpan) dependency.ConnectionSeries {
	series := dependency.ConnectionSeries{Connects: 1}
	series.SrcPodUID = s.SrcPodUID
	series.SrcNamespace = s.SrcNamespace
	series.SrcPod = s.SrcPod
	series.DstPodUID = s.DstPodUID
	series.DstNamespace = s.DstNamespace
	series.DstPod = s.DstPod
	series.DstIP = s.DstIP
	series.DstPort = s.DstPort
	series.ActualDstIP = s.ActualDstIP
	series.ActualDstPort = s.ActualDstPort
	series.Protocol = s.Protocol
	series.DstHostname = s.DstHostname
	return series
}
