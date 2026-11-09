package server

import (
	"context"
	"expvar"
	"net"
	"net/http"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	defaultMetricsPath = "/metrics"
	defaultDebugPath   = "/debug/vars"
)

// newHTTPServer builds the HTTP server: the grpc-gateway mux (dialing the gRPC server on the listener's own address),
// metrics and debug endpoints, custom handlers and middlewares.
func (s *Server) newHTTPServer(ctx context.Context, addr net.Addr) (*http.Server, error) {
	// configure mux options
	muxOpts := []runtime.ServeMuxOption{
		runtime.WithMarshalerOption(
			runtime.MIMEWildcard,
			&runtime.JSONPb{
				MarshalOptions: protojson.MarshalOptions{
					UseProtoNames:   true,
					UseEnumNumbers:  true,
					EmitUnpopulated: true,
				},
				UnmarshalOptions: protojson.UnmarshalOptions{
					DiscardUnknown: true,
				},
			},
		),
	}

	mux := runtime.NewServeMux(muxOpts...)

	// register handler
	if s.register.http != nil {
		opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
		if err := s.register.http(ctx, mux, dialAddress(addr), opts); err != nil {
			return nil, err
		}
	}

	mux.HandlePath(http.MethodGet, defaultDebugPath, func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		expvar.Handler().ServeHTTP(w, r)
	})
	mux.HandlePath(http.MethodGet, defaultMetricsPath, func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		promhttp.Handler().ServeHTTP(w, r)
	})

	// add middlewares
	middlewares := Chain(s.config.httpServerMiddlewares...)
	handler := middlewares(mux)

	// serve http server mux handlers
	httpMux := http.NewServeMux()
	for _, h := range s.config.httpServerMuxHandlers {
		h(httpMux)
	}
	httpMux.Handle("/", handler)

	// serve http server handlers
	forwardMux := http.NewServeMux()
	for path, handler := range s.config.httpServerHandlers {
		forwardMux.HandleFunc(path, handler)
		httpMux.Handle(path, middlewares(forwardMux))
	}

	return &http.Server{Handler: httpMux, ReadHeaderTimeout: 10 * time.Second}, nil
}

// dialAddress is where the gateway reaches the gRPC server: the listener's own address, with an unspecified host
// (0.0.0.0 or ::) replaced by loopback.
func dialAddress(addr net.Addr) string {
	host, port, err := net.SplitHostPort(addr.String())
	if err != nil {
		return addr.String()
	}
	if ip := net.ParseIP(host); ip == nil || ip.IsUnspecified() {
		host = "127.0.0.1"
	}
	return net.JoinHostPort(host, port)
}
