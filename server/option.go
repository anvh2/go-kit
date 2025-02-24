package server

import (
	"google.golang.org/grpc"
)

type Option func(*Config)

type Config struct {
	port int

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
		port: 8080,
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

func WithGRPCServerOptions(options ...grpc.ServerOption) Option {
	return func(c *Config) {
		c.grpcServerOptions = append(c.grpcServerOptions, options...)
	}
}

func WithGRPCUnaryInterceptor(interceptor grpc.UnaryServerInterceptor) Option {
	return func(c *Config) {
		c.grpcUnaryInterceptors = append(c.grpcUnaryInterceptors, interceptor)
	}
}

func WithGRPCStreamInterceptor(interceptor grpc.StreamServerInterceptor) Option {
	return func(c *Config) {
		c.grpcStreamInterceptors = append(c.grpcStreamInterceptors, interceptor)
	}
}

func WithHTTPServerHandler(handlers map[string]HTTPServerHandler) Option {
	return func(c *Config) {
		c.httpServerHandlers = handlers
	}
}

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
