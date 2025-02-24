package server

import (
	"context"
	"expvar"
	"fmt"
	"net"
	"net/http"

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

// serve http request
func (s *Server) httpServe(ctx context.Context, l net.Listener) error {
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
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	endPoint := fmt.Sprintf("0.0.0.0:%d", s.config.port)

	if err := s.register.http(ctx, mux, endPoint, opts); err != nil {
		return err
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

	s.server.http = &http.Server{
		Handler: httpMux,
	}
	return s.server.http.Serve(l)
}
