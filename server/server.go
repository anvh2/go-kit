package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/soheilhy/cmux"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

// RegisterGRPCHandlerFunc register server from
type RegisterGRPCHandlerFunc func(s *grpc.Server)

// RegisterHTTPHandlerFunc ...
type RegisterHTTPHandlerFunc func(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) (err error)

// HTTPServerHandler ...
type HTTPServerHandler func(http.ResponseWriter, *http.Request)

// HTTPServerMuxHandler ...
type HTTPServerMuxHandler func(*http.ServeMux)

// HTTPServerMiddleware ...
type HTTPServerMiddleware func(http.Handler) http.Handler

type Server struct {
	config *Config

	server *struct {
		grpc *grpc.Server
		http *http.Server
	}

	register *struct {
		grpc RegisterGRPCHandlerFunc
		http RegisterHTTPHandlerFunc
	}
}

func New(opts ...Option) *Server {
	config := evaluateOptions(opts)
	return &Server{
		config: config,
		server: &struct {
			grpc *grpc.Server
			http *http.Server
		}{},
		register: &struct {
			grpc RegisterGRPCHandlerFunc
			http RegisterHTTPHandlerFunc
		}{},
	}
}

func (s *Server) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.config.port))
	if err != nil {
		return err
	}

	// catch sig
	sigs := make(chan os.Signal, 1)
	done := make(chan error, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		sig := <-sigs
		fmt.Println("Exiting...: ", sig)

		s.server.grpc.Stop()
		s.server.http.Close()

		cancel()
		close(done)
	}()

	go s.serve(ctx, lis)

	fmt.Println("Server now listening at: " + lis.Addr().String())
	fmt.Println("Ctrl-C to interrupt...")

	err = <-done
	for _, hook := range s.config.shutdownHooks {
		hook()
	}

	fmt.Println("Shutted down.", zap.Error(err))
	return err
}

// start listening grpc & http & exporter request
func (s *Server) serve(ctx context.Context, listener net.Listener) {
	m := cmux.New(listener)
	grpcListener := m.MatchWithWriters(cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"))
	httpListener := m.Match(cmux.HTTP1Fast())

	g := new(errgroup.Group)
	g.Go(func() error { return s.grpcServe(ctx, grpcListener) })
	g.Go(func() error { return s.httpServe(ctx, httpListener) })
	g.Go(func() error { return m.Serve() })

	g.Wait()
}
