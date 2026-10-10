package relay

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nikhil-thorat/relay/internal/target"
)

func TestAdminTargets(t *testing.T) {
	pool := target.NewPool()
	_ = pool.Add(&target.Target{ID: "api_1", Address: "localhost:9001", Weight: 1})
	_ = pool.SetHealthy("api_1", true)

	h := buildHandler(pool, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))

	req := httptest.NewRequest(http.MethodGet, "/admin/targets", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var out []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(out) != 1 {
		t.Fatalf("expected 1 target, got %d", len(out))
	}
}

func TestHealthz(t *testing.T) {
	pool := target.NewPool()
	h := buildHandler(pool, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestReadyz(t *testing.T) {
	pool := target.NewPool()
	_ = pool.Add(&target.Target{ID: "api_1", Address: "localhost:9001"})
	h := buildHandler(pool, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when none healthy, got %d", rec.Code)
	}

	_ = pool.SetHealthy("api_1", true)
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req)
	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200 when healthy, got %d", rec2.Code)
	}
}
