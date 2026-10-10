package span

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/thread_koder/mochi/agent/internal/logger"
	"github.com/thread_koder/mochi/agent/internal/metrics"
)

const (
	SourceMochiEBPF = "mochi-ebpf"

	RPCSystemHTTP = "http"
	RPCSystemGRPC = "grpc"

	SampledByHead  = "head"
	SampledByError = "error"

	queueCap    = 8192
	flushSize   = 256
	flushEvery  = time.Second
	postTimeout = 10 * time.Second
)

type Hop struct {
	TraceID      [TraceIDSize]byte
	SpanID       [SpanIDSize]byte
	ParentSpanID [SpanIDSize]byte
	Start        time.Time
	End          time.Time
	Method       string
	Route        string
	StatusClass  string
	StatusCode   int
	GRPC         bool
	GRPCStatus   *int
	SampledBy    string
	Key          metrics.SeriesKey
}

type exportSpan struct {
	TraceID       string    `json:"trace_id"`
	SpanID        string    `json:"span_id"`
	ParentSpanID  *string   `json:"parent_span_id"`
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

type exportRequest struct {
	Spans []exportSpan `json:"spans"`
}

// Exporter queues sampled hops and POSTs them to core asynchronously.
type Exporter struct {
	url    string
	client *http.Client
	wake   chan struct{}

	mu      sync.Mutex
	queue   []Hop
	pending []Hop
}

func NewExporter(coreURL string) *Exporter {
	coreURL = strings.TrimRight(strings.TrimSpace(coreURL), "/")
	if coreURL == "" {
		return nil
	}
	return &Exporter{
		url: coreURL + "/api/v1/ingest/spans",
		client: &http.Client{
			Timeout: postTimeout,
		},
		wake:  make(chan struct{}, 1),
		queue: make([]Hop, 0, flushSize),
	}
}

// Offer enqueues a hop. Drops new when the queue is full. Never blocks.
func (e *Exporter) Offer(hop Hop) {
	if e == nil {
		return
	}
	e.mu.Lock()
	if len(e.queue) >= queueCap {
		e.mu.Unlock()
		return
	}
	e.queue = append(e.queue, hop)
	full := len(e.queue) >= flushSize
	e.mu.Unlock()
	if full {
		select {
		case e.wake <- struct{}{}:
		default:
		}
	}
}

func (e *Exporter) Run(ctx context.Context) {
	if e == nil {
		return
	}
	log := logger.WithComponent("span-exporter")
	ticker := time.NewTicker(flushEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// ctx is already cancelled. Background lets the last POST finish under the client timeout.
			e.flush(context.Background(), log)
			return
		case <-ticker.C:
			e.flush(ctx, log)
		case <-e.wake:
			e.flush(ctx, log)
		}
	}
}

func (e *Exporter) flush(ctx context.Context, log zerolog.Logger) {
	e.mu.Lock()
	if len(e.pending) == 0 && len(e.queue) == 0 {
		e.mu.Unlock()
		return
	}
	if len(e.pending) == 0 {
		batchSize := min(len(e.queue), flushSize)
		e.pending = append(e.pending[:0], e.queue[:batchSize]...)
		e.queue = e.queue[batchSize:]
	}
	batch := e.pending
	e.mu.Unlock()

	if err := e.post(ctx, batch); err != nil {
		log.Error().Err(err).Msg("Failed to export spans")
		return
	}

	e.mu.Lock()
	e.pending = e.pending[:0]
	e.mu.Unlock()
}

func (e *Exporter) post(ctx context.Context, batch []Hop) error {
	body := exportRequest{Spans: make([]exportSpan, 0, len(batch))}
	for _, hop := range batch {
		rpcSystem := RPCSystemHTTP
		if hop.GRPC {
			rpcSystem = RPCSystemGRPC
		}
		var parentHex *string
		if !IsZeroSpanID(hop.ParentSpanID) {
			encoded := hex.EncodeToString(hop.ParentSpanID[:])
			parentHex = &encoded
		}
		body.Spans = append(body.Spans, exportSpan{
			TraceID:       hex.EncodeToString(hop.TraceID[:]),
			SpanID:        hex.EncodeToString(hop.SpanID[:]),
			ParentSpanID:  parentHex,
			StartAt:       hop.Start.UTC(),
			EndAt:         hop.End.UTC(),
			Method:        hop.Method,
			Route:         hop.Route,
			StatusClass:   hop.StatusClass,
			StatusCode:    hop.StatusCode,
			GRPCStatus:    hop.GRPCStatus,
			RPCSystem:     rpcSystem,
			Source:        SourceMochiEBPF,
			SampledBy:     hop.SampledBy,
			SrcPodUID:     hop.Key.SrcPodUID,
			SrcNamespace:  hop.Key.SrcNamespace,
			SrcPod:        hop.Key.SrcPod,
			DstPodUID:     hop.Key.DstPodUID,
			DstNamespace:  hop.Key.DstNamespace,
			DstPod:        hop.Key.DstPod,
			DstIP:         hop.Key.DstIP,
			DstPort:       hop.Key.DstPort,
			ActualDstIP:   hop.Key.ActualDstIP,
			ActualDstPort: hop.Key.ActualDstPort,
			Protocol:      hop.Key.Protocol,
			DstHostname:   hop.Key.DstHostname,
		})
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal spans: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to build span export request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to post spans: %w", err)
	}
	defer resp.Body.Close()
	// Drain so the keep-alive connection can be reused.
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode == http.StatusBadRequest {
		// Drop bad batch so a poison payload cannot block the queue forever.
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("failed to post spans: status %d", resp.StatusCode)
	}
	return nil
}
