package relay

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikhil-thorat/relay/internal/config"
)

func TestSmoke(t *testing.T) {
	backend := httptest.NewServer(
		http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(
				[]byte("relay works"),
			)
		}),
	)

	defer backend.Close()

	address := strings.TrimPrefix(
		backend.URL,
		"http://",
	)

	cfg, err := config.Load(
		"../../testdata/smoke.yml",
	)
	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	cfg.Targets[0].Address = address

	relay, err := New(
		cfg,
		prometheus.NewRegistry(),
	)
	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	go func() {
		_ = relay.Start()
	}()

	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get(
		"http://localhost:18080",
	)
	if err != nil {
		t.Fatalf(
			"failed to connect to relay: %v",
			err,
		)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf(
			"expected %d, got %d",
			http.StatusOK,
			resp.StatusCode,
		)
	}

	resp, err = http.Get(
		"http://localhost:19090/metrics",
	)
	if err != nil {
		t.Fatalf(
			"failed to connect to metrics: %v",
			err,
		)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf(
			"expected %d, got %d",
			http.StatusOK,
			resp.StatusCode,
		)
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Second,
	)
	defer cancel()

	if err := relay.Shutdown(ctx); err != nil {
		t.Fatalf(
			"unexpected shutdown error: %v",
			err,
		)
	}
}
