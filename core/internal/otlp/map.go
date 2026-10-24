package otlp

import (
	"context"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/thread_koder/mochi/core/internal/dependency"
	"github.com/thread_koder/mochi/core/internal/span"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

const (
	attrK8sPodUID        = "k8s.pod.uid"
	attrK8sPodName       = "k8s.pod.name"
	attrK8sNamespace     = "k8s.namespace.name"
	attrHTTPMethod       = "http.request.method"
	attrHTTPMethodLegacy = "http.method"
	attrHTTPRoute        = "http.route"
	attrHTTPStatus       = "http.response.status_code"
	attrHTTPStatusLegacy = "http.status_code"
	attrURLPath          = "url.path"
	attrHTTPTarget       = "http.target"
	attrRPCSystem        = "rpc.system"
	attrRPCService       = "rpc.service"
	attrRPCMethod        = "rpc.method"
	attrRPCGRPCStatus    = "rpc.grpc.status_code"
	attrDBSystem         = "db.system"
	attrDBOperation      = "db.operation"
	attrDBName           = "db.name"
	attrDBCollection     = "db.collection.name"
	attrDBStatement      = "db.statement"
	attrServerAddress    = "server.address"
	attrServerPort       = "server.port"
	attrNetworkPeerAddr  = "network.peer.address"
	attrNetworkPeerPort  = "network.peer.port"
	attrNetPeerName      = "net.peer.name"
	attrNetPeerIP        = "net.peer.ip"
	attrNetPeerPort      = "net.peer.port"
	rpcSystemGRPCValue   = "grpc"
	maxMethodLen         = 32
	maxRouteLen          = 128
	maxRejectReasons     = 3

	// google/rpc Code values (code.proto).
	grpcOK                 = 0
	grpcCancelled          = 1
	grpcUnknown            = 2
	grpcInvalidArgument    = 3
	grpcDeadlineExceeded   = 4
	grpcNotFound           = 5
	grpcAlreadyExists      = 6
	grpcPermissionDenied   = 7
	grpcResourceExhausted  = 8
	grpcFailedPrecondition = 9
	grpcAborted            = 10
	grpcOutOfRange         = 11
	grpcUnimplemented      = 12
	grpcInternal           = 13
	grpcUnavailable        = 14
	grpcDataLoss           = 15
	grpcUnauthenticated    = 16
)

type mapResult struct {
	Spans    []span.IngestSpan
	Rejected int64
	Message  string
}

func mapTraces(ctx context.Context, traces ptrace.Traces) mapResult {
	var result mapResult
	var rejectReasons []string
	cache := span.NewPodIdentityCache()

	for _, resourceSpans := range traces.ResourceSpans().All() {
		resource := resourceSpans.Resource().Attributes()
		for _, scopeSpans := range resourceSpans.ScopeSpans().All() {
			for _, traceSpan := range scopeSpans.Spans().All() {
				mapped, err := mapSpan(ctx, cache, resource, traceSpan)
				if err != nil {
					result.Rejected++
					if len(rejectReasons) < maxRejectReasons {
						rejectReasons = append(rejectReasons, err.Error())
					}
					continue
				}
				result.Spans = append(result.Spans, mapped)
			}
		}
	}
	if result.Rejected > 0 {
		result.Message = fmt.Sprintf("rejected %d span(s): %s", result.Rejected, strings.Join(rejectReasons, ", "))
	}
	return result
}

func mapSpan(ctx context.Context, cache *span.PodIdentityCache, resource pcommon.Map, traceSpan ptrace.Span) (span.IngestSpan, error) {
	traceID := traceSpan.TraceID()
	spanID := traceSpan.SpanID()
	if traceID.IsEmpty() {
		return span.IngestSpan{}, fmt.Errorf("empty trace_id")
	}
	if spanID.IsEmpty() {
		return span.IngestSpan{}, fmt.Errorf("empty span_id")
	}

	start := traceSpan.StartTimestamp().AsTime()
	end := traceSpan.EndTimestamp().AsTime()
	if start.IsZero() || end.IsZero() {
		return span.IngestSpan{}, fmt.Errorf("missing start or end time")
	}
	if end.Before(start) {
		return span.IngestSpan{}, fmt.Errorf("end before start")
	}

	attrs := traceSpan.Attributes()
	uid := attrStr(resource, attrK8sPodUID)
	if uid == "" {
		uid = attrStr(attrs, attrK8sPodUID)
	}
	namespace := attrStr(resource, attrK8sNamespace)
	if namespace == "" {
		namespace = attrStr(attrs, attrK8sNamespace)
	}
	podName := attrStr(resource, attrK8sPodName)
	if podName == "" {
		podName = attrStr(attrs, attrK8sPodName)
	}
	id, err := span.FillPodIdentity(ctx, cache, uid, namespace, podName)
	if err != nil {
		return span.IngestSpan{}, err
	}

	kind := mapKind(traceSpan.Kind())
	rpcSystem, method, route, statusCode, statusClass, grpcStatus := mapOperation(traceSpan, attrs)

	out := span.IngestSpan{
		TraceID:      hex.EncodeToString(traceID[:]),
		SpanID:       hex.EncodeToString(spanID[:]),
		StartAt:      start,
		EndAt:        end,
		Method:       truncate(method, maxMethodLen),
		Route:        truncate(route, maxRouteLen),
		StatusClass:  statusClass,
		StatusCode:   statusCode,
		GRPCStatus:   grpcStatus,
		RPCSystem:    rpcSystem,
		Kind:         kind,
		Source:       span.SourceOTLP,
		SampledBy:    span.SampledByHead,
		SrcPodUID:    id.UID,
		SrcNamespace: id.Namespace,
		SrcPod:       id.Name,
		Protocol:     dependency.ProtocolTCP,
	}

	if parent := traceSpan.ParentSpanID(); !parent.IsEmpty() {
		encoded := hex.EncodeToString(parent[:])
		out.ParentSpanID = &encoded
	}

	if kind == span.KindClient || kind == span.KindProducer {
		fillDest(&out, attrs)
	}

	return out, nil
}

func mapKind(kind ptrace.SpanKind) string {
	switch kind {
	case ptrace.SpanKindInternal:
		return span.KindInternal
	case ptrace.SpanKindServer:
		return span.KindServer
	case ptrace.SpanKindClient:
		return span.KindClient
	case ptrace.SpanKindProducer:
		return span.KindProducer
	case ptrace.SpanKindConsumer:
		return span.KindConsumer
	default:
		return span.KindUnspecified
	}
}

func mapOperation(traceSpan ptrace.Span, attrs pcommon.Map) (rpcSystem, method, route string, statusCode int, statusClass string, grpcStatus *int) {
	spanError := traceSpan.Status().Code() == ptrace.StatusCodeError

	if attrStr(attrs, attrRPCSystem) == rpcSystemGRPCValue {
		service := attrStr(attrs, attrRPCService)
		rpcMethod := attrStr(attrs, attrRPCMethod)
		route = "/" + service + "/" + rpcMethod
		if service == "" && rpcMethod == "" {
			route = traceSpan.Name()
		}
		method = "POST"
		code, ok := attrInt(attrs, attrRPCGRPCStatus)
		if !ok {
			if spanError {
				code = grpcUnknown
			} else {
				code = grpcOK
			}
		}
		grpcStatus = &code
		statusCode, statusClass = ensureHTTPStatus(grpcStatusHTTP(code), spanError)
		return span.RPCSystemGRPC, method, route, statusCode, statusClass, grpcStatus
	}

	httpMethod := attrStr(attrs, attrHTTPMethod, attrHTTPMethodLegacy)
	if httpMethod != "" || attrStr(attrs, attrHTTPRoute, attrURLPath, attrHTTPTarget) != "" {
		method = httpMethod
		if method == "" {
			method = "GET"
		}
		route = attrStr(attrs, attrHTTPRoute)
		if route == "" {
			route = pathWithoutQuery(attrStr(attrs, attrURLPath, attrHTTPTarget))
		}
		if route == "" {
			route = traceSpan.Name()
		}
		if code, ok := attrInt(attrs, attrHTTPStatus, attrHTTPStatusLegacy); ok {
			statusCode = code
		} else if spanError {
			statusCode = http.StatusInternalServerError
		} else {
			statusCode = http.StatusOK
		}
		statusCode, statusClass = ensureHTTPStatus(statusCode, spanError)
		return span.RPCSystemHTTP, method, route, statusCode, statusClass, nil
	}

	if dbSystem := attrStr(attrs, attrDBSystem); dbSystem != "" {
		method = attrStr(attrs, attrDBOperation)
		if method == "" {
			method = dbSystem
		}
		route = attrStr(attrs, attrDBName, attrDBCollection, attrDBStatement)
		if route == "" {
			route = traceSpan.Name()
		}
		statusCode, statusClass = ensureHTTPStatus(0, spanError)
		return span.RPCSystemDB, method, route, statusCode, statusClass, nil
	}

	method = traceSpan.Name()
	if method == "" {
		method = "unknown"
	}
	route = traceSpan.Name()
	statusCode, statusClass = ensureHTTPStatus(0, spanError)
	return span.RPCSystemOther, method, route, statusCode, statusClass, nil
}

func ensureHTTPStatus(statusCode int, spanError bool) (int, string) {
	if class := statusClass(statusCode); class != "" {
		return statusCode, class
	}
	if spanError {
		return http.StatusInternalServerError, "5xx"
	}
	return http.StatusOK, "2xx"
}

func fillDest(out *span.IngestSpan, attrs pcommon.Map) {
	host := attrStr(attrs, attrServerAddress, attrNetworkPeerAddr, attrNetPeerName, attrNetPeerIP)
	port, _ := attrInt(attrs, attrServerPort, attrNetworkPeerPort, attrNetPeerPort)
	if host == "" {
		return
	}
	if parsedHost, parsedPort := parseHostPort(host); parsedHost != "" {
		host = parsedHost
		if port == 0 && parsedPort != 0 {
			port = parsedPort
		}
	}
	if isIP(host) {
		out.DstIP = host
		out.ActualDstIP = host
		out.DstPort = port
		out.ActualDstPort = port
		return
	}
	out.DstHostname = host
	out.DstPort = port
	out.ActualDstPort = port
}

func attrStr(attributes pcommon.Map, keys ...string) string {
	for _, key := range keys {
		if value, ok := attributes.Get(key); ok {
			text := strings.TrimSpace(value.AsString())
			if text != "" {
				return text
			}
		}
	}
	return ""
}

func attrInt(attributes pcommon.Map, keys ...string) (int, bool) {
	for _, key := range keys {
		value, ok := attributes.Get(key)
		if !ok {
			continue
		}
		switch value.Type() {
		case pcommon.ValueTypeInt:
			return int(value.Int()), true
		case pcommon.ValueTypeStr:
			parsed, err := strconv.Atoi(strings.TrimSpace(value.Str()))
			if err == nil {
				return parsed, true
			}
		}
	}
	return 0, false
}

func truncate(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen]
}

func pathWithoutQuery(path string) string {
	if before, _, ok := strings.Cut(path, "?"); ok {
		return before
	}
	return path
}

func statusClass(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 300 && code < 400:
		return "3xx"
	case code >= 400 && code < 500:
		return "4xx"
	case code >= 500 && code < 600:
		return "5xx"
	default:
		return ""
	}
}

func grpcStatusHTTP(code int) int {
	switch code {
	case grpcOK:
		return http.StatusOK
	case grpcCancelled:
		return 499
	case grpcUnknown:
		return http.StatusInternalServerError
	case grpcInvalidArgument:
		return http.StatusBadRequest
	case grpcDeadlineExceeded:
		return http.StatusGatewayTimeout
	case grpcNotFound:
		return http.StatusNotFound
	case grpcAlreadyExists:
		return http.StatusConflict
	case grpcPermissionDenied:
		return http.StatusForbidden
	case grpcResourceExhausted:
		return http.StatusTooManyRequests
	case grpcFailedPrecondition:
		return http.StatusBadRequest
	case grpcAborted:
		return http.StatusConflict
	case grpcOutOfRange:
		return http.StatusBadRequest
	case grpcUnimplemented:
		return http.StatusNotImplemented
	case grpcInternal:
		return http.StatusInternalServerError
	case grpcUnavailable:
		return http.StatusServiceUnavailable
	case grpcDataLoss:
		return http.StatusInternalServerError
	case grpcUnauthenticated:
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}

func parseHostPort(addr string) (host string, port int) {
	addr = strings.TrimSpace(addr)
	if host, portStr, err := net.SplitHostPort(addr); err == nil {
		port, _ = strconv.Atoi(portStr)
		return host, port
	}
	return addr, 0
}

func isIP(text string) bool {
	return net.ParseIP(text) != nil
}
