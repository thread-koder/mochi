package otlp

import (
	"compress/gzip"
	"context"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/thread_koder/mochi/core/internal/logger"
	"github.com/thread_koder/mochi/core/internal/span"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
	rpcstatus "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const (
	contentTypeProtobuf = "application/x-protobuf"
	contentTypeJSON     = "application/json"
	contentEncodingGzip = "gzip"
	tracesPath          = "/v1/traces"
)

var errBodyTooLarge = errors.New("body too large")

type httpHandler struct {
	maxBytes     int
	writeTimeout time.Duration
}

func (handler *httpHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	log := logger.WithComponent("otlp")

	contentType, ok := parseOTLPContentType(request.Header.Get("Content-Type"))
	if !ok {
		writeOTLPStatus(writer, http.StatusUnsupportedMediaType, contentTypeJSON, "unsupported Content-Type")
		return
	}

	if request.Method != http.MethodPost {
		writeOTLPStatus(writer, http.StatusMethodNotAllowed, contentType, "method not allowed")
		return
	}

	body, err := readBody(writer, request, handler.maxBytes)
	if err != nil {
		if errors.Is(err, errBodyTooLarge) {
			writeOTLPStatus(writer, http.StatusRequestEntityTooLarge, contentType, "request body too large")
			return
		}
		log.Error().Err(err).Msg("Failed to read OTLP body")
		writeOTLPStatus(writer, http.StatusBadRequest, contentType, "invalid request body")
		return
	}

	req := ptraceotlp.NewExportRequest()
	if len(body) > 0 {
		switch contentType {
		case contentTypeProtobuf:
			err = req.UnmarshalProto(body)
		case contentTypeJSON:
			err = req.UnmarshalJSON(body)
		}
		if err != nil {
			writeOTLPStatus(writer, http.StatusBadRequest, contentType, "failed to decode ExportTraceServiceRequest")
			return
		}
	}

	ctx, cancel := context.WithTimeout(request.Context(), handler.writeTimeout)
	defer cancel()

	resp, err := ingestTraces(ctx, req.Traces())
	if err != nil {
		log.Error().Err(err).Msg("OTLP HTTP ingest failed")
		if _, ok := errors.AsType[*span.InvalidError](err); ok {
			writeOTLPStatus(writer, http.StatusBadRequest, contentType, "ingest failed")
			return
		}
		writeOTLPStatus(writer, http.StatusServiceUnavailable, contentType, "ingest failed")
		return
	}

	var payload []byte
	switch contentType {
	case contentTypeProtobuf:
		payload, err = resp.MarshalProto()
	case contentTypeJSON:
		payload, err = resp.MarshalJSON()
	}
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal OTLP response")
		writeOTLPStatus(writer, http.StatusInternalServerError, contentType, "failed to marshal response")
		return
	}

	writer.Header().Set("Content-Type", contentType)
	writer.WriteHeader(http.StatusOK)
	if _, err := writer.Write(payload); err != nil {
		log.Error().Err(err).Msg("Failed to write OTLP response")
		return
	}
}

func parseOTLPContentType(header string) (string, bool) {
	mediaType, _, err := mime.ParseMediaType(header)
	if err != nil {
		return "", false
	}
	switch mediaType {
	case contentTypeProtobuf, contentTypeJSON:
		return mediaType, true
	default:
		return "", false
	}
}

func readBody(writer http.ResponseWriter, request *http.Request, maxBytes int) ([]byte, error) {
	defer request.Body.Close()

	var reader io.Reader
	if strings.EqualFold(request.Header.Get("Content-Encoding"), contentEncodingGzip) {
		gzipReader, err := gzip.NewReader(request.Body)
		if err != nil {
			return nil, err
		}
		defer gzipReader.Close()
		reader = io.LimitReader(gzipReader, int64(maxBytes)+1)
	} else {
		request.Body = http.MaxBytesReader(writer, request.Body, int64(maxBytes))
		reader = request.Body
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			return nil, errBodyTooLarge
		}
		return nil, err
	}
	if len(data) > maxBytes {
		return nil, errBodyTooLarge
	}
	return data, nil
}

func writeOTLPStatus(writer http.ResponseWriter, httpCode int, contentType, message string) {
	status := &rpcstatus.Status{
		Code:    int32(httpToRPCCode(httpCode)),
		Message: message,
	}
	writer.Header().Set("Content-Type", contentType)
	writer.WriteHeader(httpCode)
	switch contentType {
	case contentTypeProtobuf:
		payload, err := proto.Marshal(status)
		if err != nil {
			return
		}
		_, _ = writer.Write(payload)
	default:
		payload, err := protojson.Marshal(status)
		if err != nil {
			return
		}
		_, _ = writer.Write(payload)
	}
}

func httpToRPCCode(httpCode int) int32 {
	switch httpCode {
	case http.StatusBadRequest, http.StatusUnsupportedMediaType:
		return grpcInvalidArgument
	case http.StatusRequestEntityTooLarge:
		return grpcResourceExhausted
	case http.StatusTooManyRequests, http.StatusServiceUnavailable:
		return grpcUnavailable
	case http.StatusInternalServerError:
		return grpcInternal
	default:
		return grpcUnknown
	}
}
