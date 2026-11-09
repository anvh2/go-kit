package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/soheilhy/cmux"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

// RegisterGRPCHandlerFunc registers gRPC services on the server.
type RegisterGRPCHandlerFunc func(s *grpc.Server)

// RegisterHTTPHandlerFunc registers grpc-gateway handlers that proxy to the gRPC endpoint.
type RegisterHTTPHandlerFunc func(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) (err error)

// HTTPServerHandler ...
type HTTPServerHandler func(http.ResponseWriter, *http.Request)

// HTTPServerMuxHandler ...
type HTTPServerMuxHandler func(*http.ServeMux)

// HTTPServerMiddleware ...
type HTTPServerMiddleware func(http.Handler) http.Handler

// shutdownTimeout bounds graceful shutdown of the gRPC and HTTP servers.
const shutdownTimeout = 10 * time.Second

type Server struct {
	config *Config

	register struct {
		grpc RegisterGRPCHandlerFunc
		http RegisterHTTPHandlerFunc
	}
}

func New(opts ...Option) *Server {
	return &Server{config: evaluateOptions(opts)}
}

// Register sets the functions that register gRPC services and their HTTP gateway handlers. Either may be nil.
func (s *Server) Register(grpcFn RegisterGRPCHandlerFunc, httpFn RegisterHTTPHandlerFunc) *Server {
	s.register.grpc = grpcFn
	s.register.http = httpFn
	return s
}

// Start listens on the configured port and serves until SIGINT or SIGTERM, then runs the shutdown hooks.
func (s *Server) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.config.port))
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Println("Server now listening at: " + lis.Addr().String())
	err = s.Serve(ctx, lis)
	for _, hook := range s.config.shutdownHooks {
		hook()
	}
	return err
}

// Serve serves gRPC and HTTP (gateway, metrics, debug and custom handlers) on one listener until ctx is cancelled or a
// server fails. On cancellation both servers shut down gracefully (bounded by shutdownTimeout) and Serve returns nil.
// The listener is closed when Serve returns. Shutdown hooks are run by Start only.
func (s *Server) Serve(ctx context.Context, lis net.Listener) error {
	grpcServer := s.newGRPCServer()
	httpServer, err := s.newHTTPServer(ctx, lis.Addr())
	if err != nil {
		lis.Close()
		return err
	}

	m := cmux.New(lis)
	grpcListener := m.MatchWithWriters(cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"))
	httpListener := m.Match(cmux.HTTP1Fast())

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error { return ignoreClosed(grpcServer.Serve(grpcListener)) })
	g.Go(func() error { return ignoreClosed(httpServer.Serve(httpListener)) })
	g.Go(func() error { return ignoreClosed(m.Serve()) })
	g.Go(func() error {
		<-gctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		stopped := make(chan struct{})
		go func() { grpcServer.GracefulStop(); close(stopped) }()
		select {
		case <-stopped:
		case <-shutdownCtx.Done():
			grpcServer.Stop()
		}
		_ = httpServer.Shutdown(shutdownCtx)
		m.Close()
		return nil
	})
	return g.Wait()
}

// ignoreClosed treats the errors servers return after a deliberate shutdown as a clean exit.
func ignoreClosed(err error) error {
	if err == nil || errors.Is(err, http.ErrServerClosed) || errors.Is(err, grpc.ErrServerStopped) || errors.Is(err, cmux.ErrListenerClosed) || errors.Is(err, net.ErrClosed) || errors.Is(err, cmux.ErrServerClosed) {
		return nil
	}
	return err
}
