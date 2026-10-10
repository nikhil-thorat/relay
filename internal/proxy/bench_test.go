package proxy

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nikhil-thorat/relay/internal/balancer"
	"github.com/nikhil-thorat/relay/internal/logging"
	"github.com/nikhil-thorat/relay/internal/strategy"
	"github.com/nikhil-thorat/relay/internal/target"
)

func BenchmarkProxy(b *testing.B) {
	backend := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)
	defer backend.Close()

	address := strings.TrimPrefix(backend.URL, "http://")

	pool := target.NewPool()
	_ = pool.Add(&target.Target{ID: "api_1", Address: address})
	_ = pool.SetHealthy("api_1", true)

	bal := balancer.New(pool, &strategy.RoundRobin{})
	m := setupMetrics()
	logger, _ := logging.New("off", "text")
	p := New(bal, m, logger, 0)

	for b.Loop() {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		p.ServeHTTP(rec, req)
	}
}
