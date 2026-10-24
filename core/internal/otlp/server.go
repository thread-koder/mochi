package otlp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/thread_koder/mochi/core/internal/config"
	"github.com/thread_koder/mochi/core/internal/logger"
	"github.com/thread_koder/mochi/core/internal/span"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	_ "google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/status"
)

type Server struct {
	cfg        *config.Config
	grpcServer *grpc.Server
	httpServer *http.Server
	wg         sync.WaitGroup
}

func NewServer(cfg *config.Config) *Server {
	server := &Server{cfg: cfg}
	if cfg.API.OTLP.GRPCPort > 0 {
		server.grpcServer = grpc.NewServer(
			grpc.MaxRecvMsgSize(cfg.API.OTLP.MaxRecvBytes),
		)
		ptraceotlp.RegisterGRPCServer(server.grpcServer, &grpcHandler{})
	}
	if cfg.API.OTLP.HTTPPort > 0 {
		mux := http.NewServeMux()
		mux.Handle(tracesPath, &httpHandler{
			maxBytes:     cfg.API.OTLP.MaxRecvBytes,
			writeTimeout: cfg.API.WriteTimeoutDuration(),
		})
		server.httpServer = &http.Server{
			Addr:         fmt.Sprintf("%s:%d", cfg.API.Host, cfg.API.OTLP.HTTPPort),
			Handler:      mux,
			ReadTimeout:  cfg.API.ReadTimeoutDuration(),
			WriteTimeout: cfg.API.WriteTimeoutDuration(),
		}
	}
	return server
}

func (server *Server) enabled() bool {
	return server.grpcServer != nil || server.httpServer != nil
}

func (server *Server) Start(errCh chan<- error) error {
	if !server.enabled() {
		return nil
	}
	log := logger.WithComponent("otlp")

	if server.grpcServer != nil {
		lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", server.cfg.API.Host, server.cfg.API.OTLP.GRPCPort))
		if err != nil {
			return fmt.Errorf("failed to listen on OTLP gRPC: %w", err)
		}
		server.wg.Go(func() {
			log.Info().
				Str("host", server.cfg.API.Host).
				Int("port", server.cfg.API.OTLP.GRPCPort).
				Msg("OTLP gRPC listening...")
			if err := server.grpcServer.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
				reportServeError(errCh, fmt.Errorf("failed to serve OTLP gRPC: %w", err))
			}
		})
	}

	if server.httpServer != nil {
		lis, err := net.Listen("tcp", server.httpServer.Addr)
		if err != nil {
			if server.grpcServer != nil {
				server.grpcServer.GracefulStop()
				server.wg.Wait()
			}
			return fmt.Errorf("failed to listen on OTLP HTTP: %w", err)
		}
		server.wg.Go(func() {
			log.Info().
				Str("host", server.cfg.API.Host).
				Int("port", server.cfg.API.OTLP.HTTPPort).
				Msg("OTLP HTTP listening...")
			if err := server.httpServer.Serve(lis); err != nil && !errors.Is(err, http.ErrServerClosed) {
				reportServeError(errCh, fmt.Errorf("failed to serve OTLP HTTP: %w", err))
			}
		})
	}

	return nil
}

// After select, shutdown is not reading errCh. A blocking send would deadlock wg.Wait.
func reportServeError(errCh chan<- error, err error) {
	select {
	case errCh <- err:
	default:
	}
}

func (server *Server) Shutdown(ctx context.Context) error {
	log := logger.WithComponent("otlp")
	log.Info().Msg("Shutting down OTLP...")

	var firstErr error
	if server.httpServer != nil {
		if err := server.httpServer.Shutdown(ctx); err != nil {
			firstErr = fmt.Errorf("failed to shutdown OTLP HTTP: %w", err)
		}
	}
	if server.grpcServer != nil {
		done := make(chan struct{})
		go func() {
			server.grpcServer.GracefulStop()
			close(done)
		}()
		select {
		case <-done:
		case <-ctx.Done():
			server.grpcServer.Stop()
			if firstErr == nil {
				firstErr = fmt.Errorf("failed to shutdown OTLP gRPC: %w", ctx.Err())
			}
		}
	}
	server.wg.Wait()
	return firstErr
}

type grpcHandler struct {
	ptraceotlp.UnimplementedGRPCServer
}

func (handler *grpcHandler) Export(ctx context.Context, req ptraceotlp.ExportRequest) (ptraceotlp.ExportResponse, error) {
	resp, err := ingestTraces(ctx, req.Traces())
	if err != nil {
		log := logger.WithComponent("otlp")
		log.Error().Err(err).Msg("OTLP gRPC ingest failed")
		if _, ok := errors.AsType[*span.InvalidError](err); ok {
			return ptraceotlp.ExportResponse{}, status.Error(codes.InvalidArgument, "ingest failed")
		}
		return ptraceotlp.ExportResponse{}, status.Error(codes.Unavailable, "ingest failed")
	}
	return resp, nil
}
