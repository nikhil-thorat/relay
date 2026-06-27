package relay

import (
	"errors"
	"net/http"
	"testing"
	"time"
)

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
