package health

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nikhil-thorat/relay/internal/target"
)

func TestCheckHealthy(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	defer server.Close()

	address := strings.TrimPrefix(
		server.URL,
		"http://",
	)

	checker := New(
		target.NewPool(),
		100*time.Millisecond,
		1*time.Second,
	)

	healthy := checker.Check(&target.Target{
		Address: address,
	})

	if !healthy {
		t.Fatal("expected target to be healthy")
	}
}

func TestCheckUnhealthy(t *testing.T) {
	checker := New(
		target.NewPool(),
		100*time.Millisecond,
		1*time.Second,
	)

	healthy := checker.Check(&target.Target{
		Address: "localhost:65535",
	})

	if healthy {
		t.Fatal("expected target to be unhealthy")
	}
}

func TestRunUpdatesState(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	defer server.Close()

	address := strings.TrimPrefix(
		server.URL,
		"http://",
	)

	pool := target.NewPool()

	_ = pool.Add(&target.Target{
		ID:      "api_1",
		Address: address,
	})

	checker := New(
		pool,
		100*time.Millisecond,
		1*time.Second,
	)

	checker.Run()

	state, err := pool.GetState("api_1")
	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if !state.Healthy {
		t.Fatal("expected target to be healthy")
	}
}

func TestStart(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	defer server.Close()

	address := strings.TrimPrefix(
		server.URL,
		"http://",
	)

	pool := target.NewPool()

	_ = pool.Add(&target.Target{
		ID:      "api_1",
		Address: address,
	})

	checker := New(
		pool,
		100*time.Millisecond,
		1*time.Second,
	)

	checker.Start()

	time.Sleep(
		50 * time.Millisecond,
	)

	state, err := pool.GetState("api_1")
	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if !state.Healthy {
		t.Fatal(
			"expected target to be healthy",
		)
	}

}
