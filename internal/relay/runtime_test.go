package relay

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikhil-thorat/relay/internal/config"
)

func newSlowBackend(delay time.Duration) *httptest.Server {
	return httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(delay)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("slow ok"))
		}),
	)
}

func TestStartHTTP(t *testing.T) {
	t.Run("starts server", func(t *testing.T) {
		server := newMockServer()

		relay := &Relay{
			server: server,
		}

		if err := relay.startHTTP(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if server.listenCalls != 1 {
			t.Fatalf(
				"expected server to start once, got %d",
				server.listenCalls,
			)
		}
	})

	t.Run("returns server error", func(t *testing.T) {
		expected := errors.New("listen failed")

		server := newMockServer()
		server.listenErr = expected

		relay := &Relay{
			server: server,
		}

		err := relay.startHTTP()

		if !errors.Is(err, expected) {
			t.Fatalf(
				"expected %v, got %v",
				expected,
				err,
			)
		}
	})

	t.Run("ignores server closed", func(t *testing.T) {
		server := newMockServer()
		server.listenErr = http.ErrServerClosed

		relay := &Relay{
			server: server,
		}

		if err := relay.startHTTP(); err != nil {
			t.Fatalf(
				"expected nil, got %v",
				err,
			)
		}
	})
}

func TestStartMetrics(t *testing.T) {
	t.Run("starts metrics server when enabled", func(t *testing.T) {
		server := newMockServer()

		relay := &Relay{
			metricsEnabled: true,
			metricsServer:  server,
		}

		relay.startMetrics()

		select {
		case <-server.called:
		case <-time.After(100 * time.Millisecond):
			t.Fatal("metrics server was not started")
		}

		if server.listenCalls != 1 {
			t.Fatalf(
				"expected 1 call, got %d",
				server.listenCalls,
			)
		}
	})

	t.Run("does not start metrics server when disabled", func(t *testing.T) {
		server := newMockServer()

		relay := &Relay{
			metricsEnabled: false,
			metricsServer:  server,
		}

		relay.startMetrics()

		if server.listenCalls != 0 {
			t.Fatalf(
				"expected 0 calls, got %d",
				server.listenCalls,
			)
		}
	})
}

func TestStart(t *testing.T) {
	main := newMockServer()
	metrics := newMockServer()

	relay := &Relay{
		server:         main,
		metricsServer:  metrics,
		metricsEnabled: true,
	}

	if err := relay.Start(); err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	select {
	case <-metrics.called:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("metrics server was not started")
	}

	if main.listenCalls != 1 {
		t.Fatalf(
			"expected main server to start once, got %d",
			main.listenCalls,
		)
	}

	if metrics.listenCalls != 1 {
		t.Fatalf(
			"expected metrics server to start once, got %d",
			metrics.listenCalls,
		)
	}
}

func TestTimeoutConfig(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Address: ":18081",
		},
		Strategy: config.StrategyConfig{
			Type: "round_robin",
		},
		Targets: []config.TargetConfig{
			{ID: "api_1", Address: "localhost:9001"},
		},
	}

	r, err := New(cfg, prometheus.NewRegistry())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	srv, ok := r.server.(*http.Server)
	if !ok {
		t.Fatal("expected *http.Server")
	}

	if srv.ReadHeaderTimeout != 5*time.Second {
		t.Fatalf("expected 5s read header timeout, got %v", srv.ReadHeaderTimeout)
	}

	if srv.ReadTimeout == 0 || srv.WriteTimeout == 0 || srv.IdleTimeout == 0 {
		t.Fatal("expected timeouts to be set")
	}

	if cfg.Server.ShutdownTimeout != 5*time.Second {
		t.Fatalf("expected 5s shutdown timeout, got %v", cfg.Server.ShutdownTimeout)
	}
}

func TestShutdownDrainsInflight(t *testing.T) {
	backend := newSlowBackend(200 * time.Millisecond)
	defer backend.Close()

	address := strings.TrimPrefix(backend.URL, "http://")

	cfg := &config.Config{
		Server:   config.ServerConfig{Address: "127.0.0.1:0"},
		Strategy: config.StrategyConfig{Type: "round_robin"},
		Targets:  []config.TargetConfig{{ID: "api_1", Address: address}},
		Health:   config.HealthConfig{Enabled: false},
		Metrics:  config.MetricsConfig{Enabled: false},
		Logging:  config.LoggingConfig{Level: "off", Format: "text"},
	}

	r, err := New(cfg, prometheus.NewRegistry())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_ = r.pool.SetHealthy("api_1", true)

	srv, ok := r.server.(*http.Server)
	if !ok {
		t.Fatal("expected *http.Server")
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	go func() {
		_ = srv.Serve(listener)
	}()

	time.Sleep(50 * time.Millisecond)

	done := make(chan int, 1)
	go func() {
		resp, err := http.Get("http://" + listener.Addr().String())
		if err != nil {
			done <- -1
			return
		}
		defer resp.Body.Close()
		done <- resp.StatusCode
	}()

	time.Sleep(50 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	shutdownDone := make(chan error, 1)
	go func() {
		shutdownDone <- r.Shutdown(ctx)
	}()

	select {
	case code := <-done:
		if code != http.StatusOK {
			t.Fatalf("expected inflight 200, got %d", code)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("inflight request did not complete")
	}

	select {
	case err := <-shutdownDone:
		if err != nil {
			t.Fatalf("unexpected shutdown error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("shutdown did not complete")
	}
}

func TestReloadSwapsPool(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "relay.yml")

	first := "server:\n  address: \":18082\"\nstrategy:\n  type: round_robin\ntargets:\n  - id: api_1\n    address: localhost:9001\nhealth:\n  enabled: false\nmetrics:\n  enabled: false\nlogging:\n  level: off\n"
	if err := os.WriteFile(path, []byte(first), 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r, err := New(cfg, prometheus.NewRegistry())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(r.pool.List()) != 1 {
		t.Fatalf("expected 1 target, got %d", len(r.pool.List()))
	}

	second := "server:\n  address: \":18082\"\nstrategy:\n  type: round_robin\ntargets:\n  - id: api_1\n    address: localhost:9001\n  - id: api_2\n    address: localhost:9002\nhealth:\n  enabled: false\nmetrics:\n  enabled: false\nlogging:\n  level: off\n"
	if err := os.WriteFile(path, []byte(second), 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := r.Reload(path); err != nil {
		t.Fatalf("unexpected reload error: %v", err)
	}

	if len(r.pool.List()) != 2 {
		t.Fatalf("expected 2 targets after reload, got %d", len(r.pool.List()))
	}
}
