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
)

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

// registerHealthGateway exposes GET /healthz, which calls the gRPC health service through the endpoint the server dials.
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

// start serves s on a loopback port and stops it when the test ends, failing the test if Serve returns an error.
func start(t *testing.T, s *Server) string {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
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
	return lis.Addr().String()
}

func healthCheck(t *testing.T, addr string) error {
	t.Helper()
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
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
	addr := start(t, s)

	if err := healthCheck(t, addr); err != nil {
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
	addr := start(t, New(WithGRPCUnaryInterceptor(deny)).Register(registerHealth, nil))

	if code := status.Code(healthCheck(t, addr)); code != codes.PermissionDenied {
		t.Fatalf("code = %v, want PermissionDenied", code)
	}
}

func TestServerOptionsAreApplied(t *testing.T) {
	rec := &recorder{}
	s := New(WithGRPCServerOptions(grpc.ChainUnaryInterceptor(rec.interceptor("option")))).Register(registerHealth, nil)
	addr := start(t, s)

	if err := healthCheck(t, addr); err != nil {
		t.Fatalf("health check: %v", err)
	}
	if got := rec.snapshot(); len(got) != 1 {
		t.Fatalf("interceptor from server option ran %d times, want 1", len(got))
	}
}

func TestGatewayCallsPassThroughInterceptors(t *testing.T) {
	rec := &recorder{}
	addr := start(t, New(WithGRPCUnaryInterceptor(rec.interceptor("gw"))).Register(registerHealth, registerHealthGateway))

	resp, err := http.Get("http://" + addr + "/healthz")
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
	addr := start(t, New(WithHTTPServerMiddleware(mw)).Register(registerHealth, nil))

	resp, err := http.Get("http://" + addr + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !seen {
		t.Fatalf("GET /metrics = %d, middleware seen = %v", resp.StatusCode, seen)
	}
}

func TestServeWithoutRegistrationsAndNilRegisterFuncs(t *testing.T) {
	addr := start(t, New())
	if code := status.Code(healthCheck(t, addr)); code != codes.Unimplemented {
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
