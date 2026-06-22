package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nikhil-thorat/relay/internal/balancer"
	"github.com/nikhil-thorat/relay/internal/strategy"
	"github.com/nikhil-thorat/relay/internal/target"
)

func TestNew(t *testing.T) {
	pool := target.NewPool()
	rr := &strategy.RoundRobin{}

	balancer := balancer.New(pool, rr)

	proxy := New(balancer)

	if proxy == nil {
		t.Fatal("expected proxy, got nil")
	}

}

func TestServeHTTPWithNoTargets(t *testing.T) {
	pool := target.NewPool()
	rr := &strategy.RoundRobin{}

	balancer := balancer.New(pool, rr)
	proxy := New(balancer)

	req := httptest.NewRequest(
		http.MethodGet,
		"/",
		nil,
	)

	rec := httptest.NewRecorder()

	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf(
			"expected %d, got %d",
			http.StatusBadGateway,
			rec.Code,
		)
	}

}

func TestServeHTTPForwardsRequest(t *testing.T) {
	backend := httptest.NewServer(
		http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(
				[]byte("hello from backend"),
			)
		}),
	)

	defer backend.Close()

	address := strings.TrimPrefix(
		backend.URL,
		"http://",
	)

	pool := target.NewPool()

	_ = pool.Add(&target.Target{
		ID:      "api_1",
		Address: address,
	})

	_ = pool.SetHealthy(
		"api_1",
		true,
	)

	rr := &strategy.RoundRobin{}
	balancer := balancer.New(pool, rr)

	proxy := New(balancer)

	req := httptest.NewRequest(
		http.MethodGet,
		"/",
		nil,
	)

	rec := httptest.NewRecorder()

	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"expected %d, got %d",
			http.StatusOK,
			rec.Code,
		)
	}

	body, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if string(body) != "hello from backend" {
		t.Fatalf(
			"expected backend response, got %q",
			string(body),
		)
	}
}
