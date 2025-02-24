package server

import "net/http"

func Chain(middlewares ...HTTPServerMiddleware) HTTPServerMiddleware {
	return func(next http.Handler) http.Handler {
		return buildChain(next, middlewares...)
	}
}

func buildChain(next http.Handler, middlewares ...HTTPServerMiddleware) http.Handler {
	if len(middlewares) == 0 {
		return next
	}
	return middlewares[0](buildChain(next, middlewares[1:]...))
}
