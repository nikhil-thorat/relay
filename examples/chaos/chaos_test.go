package chaos

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikhil-thorat/relay/internal/balancer"
	"github.com/nikhil-thorat/relay/internal/logging"
	"github.com/nikhil-thorat/relay/internal/metrics"
	"github.com/nikhil-thorat/relay/internal/proxy"
	"github.com/nikhil-thorat/relay/internal/strategy"
	"github.com/nikhil-thorat/relay/internal/target"
)

func TestChaosKillsBackend(t *testing.T) {
	good := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("good"))
		}),
	)
	defer good.Close()

	flaky := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("flaky"))
		}),
	)

	flakyAddr := strings.TrimPrefix(flaky.URL, "http://")
	goodAddr := strings.TrimPrefix(good.URL, "http://")

	pool := target.NewPool()
	_ = pool.Add(&target.Target{ID: "flaky", Address: flakyAddr})
	_ = pool.Add(&target.Target{ID: "good", Address: goodAddr})
	_ = pool.SetHealthy("flaky", true)
	_ = pool.SetHealthy("good", true)

	b := balancer.New(pool, &strategy.RoundRobin{})
	m := metrics.New(prometheus.NewRegistry())
	logger, _ := logging.New("off", "text")
	p := proxy.New(b, m, logger, 1)

	flaky.Close()

	var mu sync.Mutex
	counts := map[int]int{}

	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			p.ServeHTTP(rec, req)
			mu.Lock()
			counts[rec.Code]++
			mu.Unlock()
		})
	}
	wg.Wait()

	if counts[500] != 0 {
		t.Fatalf("expected zero 500s, got %v", counts)
	}

	if counts[200] == 0 {
		t.Fatalf("expected mostly 200s, got %v", counts)
	}
}
