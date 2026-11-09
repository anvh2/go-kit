package server

import (
	"context"
	"io"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

// The runtime verifier and some CI sandboxes deny all sockets (loopback included), so the server is served on an
// in-memory listener and every client dials it through the same listener.

type recorder struct {
	mu    sync.Mutex
	calls []string
}

func (r *recorder) interceptor(name string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		r.mu.Lock()
		r.calls = append(r.calls, name+" "+info.FullMethod)
		r.mu.Unlock()
		return handler(ctx, req)
	}
}

func (r *recorder) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.calls...)
}

func registerHealth(s *grpc.Server) { healthpb.RegisterHealthServer(s, health.NewServer()) }

// healthGateway exposes GET /healthz, which calls the gRPC health service with the dial options the server passes. The
// in-memory listener replaces the network address (dialAddress has its own test).
func healthGateway(lis *bufconn.Listener) RegisterHTTPHandlerFunc {
	return func(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
		return registerHealthGateway(ctx, mux, "passthrough:///bufnet", append(opts, dialer(lis)))
	}
}

func registerHealthGateway(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	conn, err := grpc.NewClient(endpoint, opts...)
	if err != nil {
		return err
	}
	go func() { <-ctx.Done(); conn.Close() }()
	client := healthpb.NewHealthClient(conn)
	return mux.HandlePath(http.MethodGet, "/healthz", func(w http.ResponseWriter, r *http.Request, _ map[string]string) {
		resp, err := client.Check(r.Context(), &healthpb.HealthCheckRequest{})
		if err != nil {
			http.Error(w, status.Code(err).String(), http.StatusForbidden)
			return
		}
		io.WriteString(w, resp.GetStatus().String())
	})
}

// start serves s on an in-memory listener and stops it when the test ends, failing the test if Serve returns an error.
func start(t *testing.T, s *Server) *bufconn.Listener {
	t.Helper()
	lis := bufconn.Listen(1 << 20)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.Serve(ctx, lis) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("Serve returned %v after cancellation", err)
			}
		case <-time.After(5 * time.Second):
			t.Error("Serve did not return after cancellation")
		}
	})
	return lis
}

func dialer(lis *bufconn.Listener) grpc.DialOption {
	return grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) })
}

func httpClient(lis *bufconn.Listener) *http.Client {
	return &http.Client{Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) { return lis.DialContext(ctx) }}}
}

func healthCheck(t *testing.T, lis *bufconn.Listener) error {
	t.Helper()
	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithTransportCredentials(insecure.NewCredentials()), dialer(lis))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = healthpb.NewHealthClient(conn).Check(ctx, &healthpb.HealthCheckRequest{})
	return err
}

func TestCustomUnaryInterceptorsRunInRegistrationOrder(t *testing.T) {
	rec := &recorder{}
	s := New(WithGRPCUnaryInterceptor(rec.interceptor("first")), WithGRPCUnaryInterceptor(rec.interceptor("second"))).Register(registerHealth, nil)
	lis := start(t, s)

	if err := healthCheck(t, lis); err != nil {
		t.Fatalf("health check: %v", err)
	}
	got := rec.snapshot()
	want := []string{"first /grpc.health.v1.Health/Check", "second /grpc.health.v1.Health/Check"}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("interceptor calls = %q, want %q", got, want)
	}
}

func TestInterceptorCanRejectACall(t *testing.T) {
	deny := func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		return nil, status.Error(codes.PermissionDenied, "denied")
	}
	lis := start(t, New(WithGRPCUnaryInterceptor(deny)).Register(registerHealth, nil))

	if code := status.Code(healthCheck(t, lis)); code != codes.PermissionDenied {
		t.Fatalf("code = %v, want PermissionDenied", code)
	}
}

func TestServerOptionsAreApplied(t *testing.T) {
	rec := &recorder{}
	s := New(WithGRPCServerOptions(grpc.ChainUnaryInterceptor(rec.interceptor("option")))).Register(registerHealth, nil)
	lis := start(t, s)

	if err := healthCheck(t, lis); err != nil {
		t.Fatalf("health check: %v", err)
	}
	if got := rec.snapshot(); len(got) != 1 {
		t.Fatalf("interceptor from server option ran %d times, want 1", len(got))
	}
}

func TestGatewayCallsPassThroughInterceptors(t *testing.T) {
	rec := &recorder{}
	var gw RegisterHTTPHandlerFunc
	s := New(WithGRPCUnaryInterceptor(rec.interceptor("gw")))
	lis := bufconn.Listen(1 << 20)
	gw = healthGateway(lis)
	s.Register(registerHealth, gw)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.Serve(ctx, lis) }()
	defer func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("Serve: %v", err)
		}
	}()

	resp, err := httpClient(lis).Get("http://bufnet/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || string(body) != "SERVING" {
		t.Fatalf("GET /healthz = %d %q, want 200 SERVING", resp.StatusCode, body)
	}
	if got := rec.snapshot(); len(got) != 1 || got[0] != "gw /grpc.health.v1.Health/Check" {
		t.Fatalf("interceptor calls = %q, want one gateway-originated Check", got)
	}
}

func TestHTTPMiddlewareAndMetricsEndpoint(t *testing.T) {
	var seen bool
	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { seen = true; next.ServeHTTP(w, r) })
	}
	lis := start(t, New(WithHTTPServerMiddleware(mw)).Register(registerHealth, nil))

	resp, err := httpClient(lis).Get("http://bufnet/metrics")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !seen {
		t.Fatalf("GET /metrics = %d, middleware seen = %v", resp.StatusCode, seen)
	}
}

func TestServeWithoutRegistrationsAndNilRegisterFuncs(t *testing.T) {
	lis := start(t, New())
	if code := status.Code(healthCheck(t, lis)); code != codes.Unimplemented {
		t.Fatalf("code = %v, want Unimplemented for an unregistered service", code)
	}
}

func TestDialAddress(t *testing.T) {
	cases := map[string]string{
		"0.0.0.0:8080":   "127.0.0.1:8080",
		"[::]:9090":      "127.0.0.1:9090",
		"127.0.0.1:7070": "127.0.0.1:7070",
		"10.1.2.3:6060":  "10.1.2.3:6060",
	}
	for in, want := range cases {
		addr, err := net.ResolveTCPAddr("tcp", in)
		if err != nil {
			t.Fatal(err)
		}
		if got := dialAddress(addr); got != want {
			t.Errorf("dialAddress(%s) = %s, want %s", in, got, want)
		}
	}
}

func TestStreamInterceptorsRun(t *testing.T) {
	var mu sync.Mutex
	var calls []string
	si := func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		mu.Lock()
		calls = append(calls, info.FullMethod)
		mu.Unlock()
		return handler(srv, ss)
	}
	lis := start(t, New(WithGRPCStreamInterceptor(si)).Register(registerHealth, nil))

	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithTransportCredentials(insecure.NewCredentials()), dialer(lis))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stream, err := healthpb.NewHealthClient(conn).Watch(ctx, &healthpb.HealthCheckRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := stream.Recv(); err != nil {
		t.Fatalf("Watch Recv: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(calls) != 1 || calls[0] != "/grpc.health.v1.Health/Watch" {
		t.Fatalf("stream interceptor calls = %q, want one Watch", calls)
	}
}

func TestPanicInCallerInterceptorIsRecovered(t *testing.T) {
	boom := func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		panic("boom")
	}
	lis := start(t, New(WithGRPCUnaryInterceptor(boom)).Register(registerHealth, nil))

	for i := 0; i < 2; i++ { // the server survives the first panic
		if code := status.Code(healthCheck(t, lis)); code != codes.Internal {
			t.Fatalf("call %d: code = %v, want Internal from the recovery interceptor", i, code)
		}
	}
}

func TestShutdownIsNotBlockedByAnIdleConnection(t *testing.T) {
	s := New().Register(registerHealth, nil)
	s.config.muxReadTimeout = 200 * time.Millisecond
	lis := bufconn.Listen(1 << 20)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.Serve(ctx, lis) }()

	idle, err := lis.Dial() // connects and never sends a byte
	if err != nil {
		t.Fatal(err)
	}
	defer idle.Close()
	time.Sleep(50 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Serve returned %v after cancellation", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Serve did not return: an idle connection blocked shutdown")
	}
}

func TestServeReturnsWhenTheListenerIsClosedExternally(t *testing.T) {
	lis := bufconn.Listen(1 << 20)
	done := make(chan error, 1)
	go func() { done <- New().Register(registerHealth, nil).Serve(context.Background(), lis) }()
	time.Sleep(50 * time.Millisecond)
	lis.Close()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("Serve returned nil, want an error for a listener closed outside shutdown")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Serve did not return after its listener was closed")
	}
}
