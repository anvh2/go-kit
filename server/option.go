package server

import (
	"time"

	"google.golang.org/grpc"
)

type Option func(*Config)

type Config struct {
	port int

	// muxReadTimeout bounds how long a new connection may take to send the bytes that identify its protocol.
	muxReadTimeout time.Duration

	grpcServerOptions      []grpc.ServerOption
	grpcUnaryInterceptors  []grpc.UnaryServerInterceptor
	grpcStreamInterceptors []grpc.StreamServerInterceptor

	httpServerHandlers    map[string]HTTPServerHandler // path -> handler
	httpServerMuxHandlers []HTTPServerMuxHandler
	httpServerMiddlewares []HTTPServerMiddleware
	shutdownHooks         []func()
}

func defaultConfig() *Config {
	return &Config{
		port:           8080,
		muxReadTimeout: 5 * time.Second,
	}
}

func evaluateOptions(opts []Option) *Config {
	options := defaultConfig()
	for _, o := range opts {
		o(options)
	}
	return options
}

func WithPort(port int) Option {
	return func(c *Config) {
		c.port = port
	}
}

// WithGRPCServerOptions adds options to the gRPC server, applied after the interceptor chain. Prefer
// WithGRPCUnaryInterceptor and WithGRPCStreamInterceptor for interceptors: an interceptor passed here as
// grpc.UnaryInterceptor or grpc.StreamInterceptor runs before the built-in recovery, metrics and validation
// interceptors, so a panic in it is not recovered.
func WithGRPCServerOptions(options ...grpc.ServerOption) Option {
	return func(c *Config) {
		c.grpcServerOptions = append(c.grpcServerOptions, options...)
	}
}

// WithGRPCUnaryInterceptor appends a unary interceptor. Caller interceptors run after the built-in ones (recovery,
// metrics, validation), in registration order, for every unary call, including calls proxied by the HTTP gateway.
// Gateway calls reach gRPC from the server's own address, so an interceptor must never trust the peer address.
func WithGRPCUnaryInterceptor(interceptor grpc.UnaryServerInterceptor) Option {
	return func(c *Config) {
		c.grpcUnaryInterceptors = append(c.grpcUnaryInterceptors, interceptor)
	}
}

// WithGRPCStreamInterceptor appends a stream interceptor; ordering is as for WithGRPCUnaryInterceptor.
func WithGRPCStreamInterceptor(interceptor grpc.StreamServerInterceptor) Option {
	return func(c *Config) {
		c.grpcStreamInterceptors = append(c.grpcStreamInterceptors, interceptor)
	}
}

// WithHTTPServerHandler serves plain HTTP handlers by path. They pass through the HTTP middlewares but not through any
// gRPC interceptor: they must enforce their own authentication.
func WithHTTPServerHandler(handlers map[string]HTTPServerHandler) Option {
	return func(c *Config) {
		c.httpServerHandlers = handlers
	}
}

// WithHTTPServerMuxHandler registers handlers directly on the root HTTP mux. They pass through neither the HTTP
// middlewares nor any gRPC interceptor: they must enforce their own authentication.
func WithHTTPServerMuxHandler(handler HTTPServerMuxHandler) Option {
	return func(c *Config) {
		c.httpServerMuxHandlers = append(c.httpServerMuxHandlers, handler)
	}
}

func WithHTTPServerMiddleware(middleware HTTPServerMiddleware) Option {
	return func(c *Config) {
		c.httpServerMiddlewares = append(c.httpServerMiddlewares, middleware)
	}
}

func WithShutdownHook(hook func()) Option {
	return func(c *Config) {
		c.shutdownHooks = append(c.shutdownHooks, hook)
	}
}
