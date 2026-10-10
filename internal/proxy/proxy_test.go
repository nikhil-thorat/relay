package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikhil-thorat/relay/internal/balancer"
	"github.com/nikhil-thorat/relay/internal/logging"
	"github.com/nikhil-thorat/relay/internal/metrics"
	"github.com/nikhil-thorat/relay/internal/strategy"
	"github.com/nikhil-thorat/relay/internal/target"
)

func setupMetrics() *metrics.Metrics {
	registry := prometheus.NewRegistry()

	return metrics.New(registry)
}

func TestNew(t *testing.T) {
	pool := target.NewPool()
	rr := &strategy.RoundRobin{}

	balancer := balancer.New(pool, rr)

	metrics := setupMetrics()

	logger, _ := logging.New("off", "text")

	proxy := New(
		balancer,
		metrics,
		logger,
		1,
	)

	if proxy == nil {
		t.Fatal("expected proxy, got nil")
	}
}

func TestServeHTTPWithNoTargets(t *testing.T) {
	pool := target.NewPool()
	rr := &strategy.RoundRobin{}

	balancer := balancer.New(pool, rr)

	metrics := setupMetrics()

	logger, _ := logging.New("off", "text")

	proxy := New(
		balancer,
		metrics,
		logger,
		1,
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/",
		nil,
	)

	rec := httptest.NewRecorder()

	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf(
			"expected %d, got %d",
			http.StatusServiceUnavailable,
			rec.Code,
		)
	}

	requests := testutil.ToFloat64(
		metrics.RequestsTotal,
	)

	if requests != 1 {
		t.Fatalf(
			"expected 1 request, got %v",
			requests,
		)
	}

	errors := testutil.ToFloat64(
		metrics.RequestErrorsTotal,
	)

	if errors != 1 {
		t.Fatalf(
			"expected 1 error, got %v",
			errors,
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

	metrics := setupMetrics()

	logger, _ := logging.New("off", "text")

	proxy := New(
		balancer,
		metrics,
		logger,
		1,
	)

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

	requests := testutil.ToFloat64(
		metrics.RequestsTotal,
	)

	if requests != 1 {
		t.Fatalf(
			"expected 1 request, got %v",
			requests,
		)
	}

	targetRequests := testutil.ToFloat64(
		metrics.TargetRequests.
			WithLabelValues("api_1", "200"),
	)

	if targetRequests != 1 {
		t.Fatalf(
			"expected 1 target request, got %v",
			targetRequests,
		)
	}

	durationCount := testutil.CollectAndCount(metrics.RequestDuration)
	if durationCount == 0 {
		t.Fatal("expected duration metric observed")
	}

	errors := testutil.ToFloat64(
		metrics.RequestErrorsTotal,
	)

	if errors != 0 {
		t.Fatalf(
			"expected 0 errors, got %v",
			errors,
		)
	}
}

func TestStatusCodes(t *testing.T) {
	t.Run("no targets returns 503", func(t *testing.T) {
		pool := target.NewPool()
		rr := &strategy.RoundRobin{}
		balancer := balancer.New(pool, rr)
		m := setupMetrics()
		logger, _ := logging.New("off", "text")
		p := New(balancer, m, logger, 1)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		p.ServeHTTP(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected %d, got %d", http.StatusServiceUnavailable, rec.Code)
		}
	})

	t.Run("bad target url returns 502", func(t *testing.T) {
		pool := target.NewPool()
		_ = pool.Add(&target.Target{ID: "bad", Address: "://bad-url"})
		_ = pool.SetHealthy("bad", true)
		rr := &strategy.RoundRobin{}
		balancer := balancer.New(pool, rr)
		m := setupMetrics()
		logger, _ := logging.New("off", "text")
		p := New(balancer, m, logger, 1)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		p.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadGateway {
			t.Fatalf("expected %d, got %d", http.StatusBadGateway, rec.Code)
		}
	})

	t.Run("backend down returns 502", func(t *testing.T) {
		pool := target.NewPool()
		_ = pool.Add(&target.Target{ID: "down", Address: "localhost:1"})
		_ = pool.SetHealthy("down", true)
		rr := &strategy.RoundRobin{}
		balancer := balancer.New(pool, rr)
		m := setupMetrics()
		logger, _ := logging.New("off", "text")
		p := New(balancer, m, logger, 1)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		p.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadGateway {
			t.Fatalf("expected %d, got %d", http.StatusBadGateway, rec.Code)
		}
	})
}

func TestHeadersForwarded(t *testing.T) {
	var gotForwarded, gotRequestID string
	backend := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotForwarded = r.Header.Get("X-Forwarded-For")
			gotRequestID = r.Header.Get("X-Request-ID")
			w.WriteHeader(http.StatusOK)
		}),
	)
	defer backend.Close()

	address := strings.TrimPrefix(backend.URL, "http://")

	pool := target.NewPool()
	_ = pool.Add(&target.Target{ID: "api_1", Address: address})
	_ = pool.SetHealthy("api_1", true)

	rr := &strategy.RoundRobin{}
	balancer := balancer.New(pool, rr)
	m := setupMetrics()
	logger, _ := logging.New("off", "text")
	p := New(balancer, m, logger, 1)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	req.Header.Set("X-Request-ID", "test-id-123")
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)

	if gotRequestID != "test-id-123" {
		t.Fatalf("expected request id passthrough, got %q", gotRequestID)
	}

	if !strings.Contains(gotForwarded, "1.2.3.4") {
		t.Fatalf("expected X-Forwarded-For to contain client ip, got %q", gotForwarded)
	}
}

func TestErrorHandlerIncrementsErrors(t *testing.T) {
	pool := target.NewPool()
	_ = pool.Add(&target.Target{ID: "down", Address: "localhost:1"})
	_ = pool.SetHealthy("down", true)

	rr := &strategy.RoundRobin{}
	balancer := balancer.New(pool, rr)
	m := setupMetrics()
	logger, _ := logging.New("off", "text")
	p := New(balancer, m, logger, 1)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected %d, got %d", http.StatusBadGateway, rec.Code)
	}

	errs := testutil.ToFloat64(m.RequestErrorsTotal)
	if errs != 1 {
		t.Fatalf("expected 1 error, got %v", errs)
	}
}

func TestConcurrentProxy(t *testing.T) {
	backend := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		}),
	)
	defer backend.Close()

	address := strings.TrimPrefix(backend.URL, "http://")

	pool := target.NewPool()
	_ = pool.Add(&target.Target{ID: "api_1", Address: address})
	_ = pool.Add(&target.Target{ID: "api_2", Address: address})
	_ = pool.SetHealthy("api_1", true)
	_ = pool.SetHealthy("api_2", true)

	rr := &strategy.RoundRobin{}
	balancer := balancer.New(pool, rr)
	m := setupMetrics()
	logger, _ := logging.New("off", "text")
	p := New(balancer, m, logger, 1)

	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			p.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Errorf("expected 200, got %d", rec.Code)
			}
		})
	}
	wg.Wait()
}

func TestRetryOnDifferentBackend(t *testing.T) {
	good := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("good"))
		}),
	)
	defer good.Close()

	goodAddr := strings.TrimPrefix(good.URL, "http://")

	pool := target.NewPool()
	_ = pool.Add(&target.Target{ID: "bad", Address: "localhost:1"})
	_ = pool.Add(&target.Target{ID: "good", Address: goodAddr})
	_ = pool.SetHealthy("bad", true)
	_ = pool.SetHealthy("good", true)

	rr := &strategy.RoundRobin{}
	b := balancer.New(pool, rr)
	m := setupMetrics()
	logger, _ := logging.New("off", "text")
	p := New(b, m, logger, 1)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected retry to succeed with 200, got %d", rec.Code)
	}
}

func TestNoRetryOnPost(t *testing.T) {
	var hits int
	good := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hits++
			w.WriteHeader(http.StatusOK)
		}),
	)
	defer good.Close()

	goodAddr := strings.TrimPrefix(good.URL, "http://")

	pool := target.NewPool()
	_ = pool.Add(&target.Target{ID: "bad", Address: "localhost:1"})
	_ = pool.Add(&target.Target{ID: "good", Address: goodAddr})
	_ = pool.SetHealthy("bad", true)
	_ = pool.SetHealthy("good", true)

	rr := &strategy.RoundRobin{}
	b := balancer.New(pool, rr)
	m := setupMetrics()
	logger, _ := logging.New("off", "text")
	p := New(b, m, logger, 1)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected POST to fail without retry (502), got %d", rec.Code)
	}

	if hits != 0 {
		t.Fatalf("expected good backend not hit on POST no-retry, hits=%d", hits)
	}
}

func TestRateLimitReturns429(t *testing.T) {
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

	rr := &strategy.RoundRobin{}
	b := balancer.New(pool, rr)
	m := setupMetrics()
	logger, _ := logging.New("off", "text")
	p := New(b, m, logger, 0)
	p.SetLimits(0, 1)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "9.9.9.9:1234"
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected first 200, got %d", rec.Code)
	}

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = "9.9.9.9:1234"
	p.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec2.Code)
	}
}

func TestMaxInflightReturns429(t *testing.T) {
	release := make(chan struct{})
	backend := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			<-release
			w.WriteHeader(http.StatusOK)
		}),
	)
	defer backend.Close()

	address := strings.TrimPrefix(backend.URL, "http://")
	pool := target.NewPool()
	_ = pool.Add(&target.Target{ID: "api_1", Address: address})
	_ = pool.SetHealthy("api_1", true)

	rr := &strategy.RoundRobin{}
	b := balancer.New(pool, rr)
	m := setupMetrics()
	logger, _ := logging.New("off", "text")
	p := New(b, m, logger, 0)
	p.SetLimits(1, 0)

	done := make(chan int, 1)
	go func() {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		p.ServeHTTP(rec, req)
		done <- rec.Code
	}()

	time.Sleep(50 * time.Millisecond)

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec2 := httptest.NewRecorder()
	p.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec2.Code)
	}

	close(release)
	if code := <-done; code != http.StatusOK {
		t.Fatalf("expected first 200, got %d", code)
	}
}
