package server

import (
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_validator "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/validator"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// newGRPCServer builds the gRPC server. Built-in interceptors run first (recovery, metrics, validation), then the caller's
// interceptors in registration order. Caller server options are applied after the interceptor chain.
func (s *Server) newGRPCServer() *grpc.Server {
	unary := append([]grpc.UnaryServerInterceptor{
		grpc_recovery.UnaryServerInterceptor(),
		grpc_prometheus.UnaryServerInterceptor,
		grpc_validator.UnaryServerInterceptor(),
	}, s.config.grpcUnaryInterceptors...)
	stream := append([]grpc.StreamServerInterceptor{
		grpc_recovery.StreamServerInterceptor(),
		grpc_prometheus.StreamServerInterceptor,
		grpc_validator.StreamServerInterceptor(),
	}, s.config.grpcStreamInterceptors...)

	opts := append([]grpc.ServerOption{
		grpc.ChainUnaryInterceptor(unary...),
		grpc.ChainStreamInterceptor(stream...),
	}, s.config.grpcServerOptions...)

	server := grpc.NewServer(opts...)
	if s.register.grpc != nil {
		s.register.grpc(server)
	}
	reflection.Register(server)

	grpc_prometheus.Register(server)
	grpc_prometheus.EnableHandlingTimeHistogram()
	return server
}
