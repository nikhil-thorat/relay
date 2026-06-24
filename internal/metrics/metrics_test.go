package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestNew(t *testing.T) {
	registry := prometheus.NewRegistry()

	metrics := New(registry)

	if metrics == nil {
		t.Fatal("expected metrics, got nil")
	}
}

func TestIncrementRequests(t *testing.T) {
	registry := prometheus.NewRegistry()

	metrics := New(registry)

	metrics.IncrementRequests()

	value := testutil.ToFloat64(
		metrics.RequestsTotal,
	)

	if value != 1 {
		t.Fatalf(
			"expected 1, got %v",
			value,
		)
	}
}

func TestIncrementErrors(t *testing.T) {
	registry := prometheus.NewRegistry()

	metrics := New(registry)

	metrics.IncrementErrors()

	value := testutil.ToFloat64(
		metrics.RequestErrorsTotal,
	)

	if value != 1 {
		t.Fatalf(
			"expected 1, got %v",
			value,
		)
	}
}

func TestSetHealthyTargets(t *testing.T) {
	registry := prometheus.NewRegistry()

	metrics := New(registry)

	metrics.SetHealthyTargets(3)

	value := testutil.ToFloat64(
		metrics.HealthyTargets,
	)

	if value != 3 {
		t.Fatalf(
			"expected 3, got %v",
			value,
		)
	}
}

func TestIncrementTargetRequests(t *testing.T) {
	registry := prometheus.NewRegistry()

	metrics := New(registry)

	metrics.IncrementTargetRequests("api_1")

	value := testutil.ToFloat64(
		metrics.TargetRequests.
			WithLabelValues("api_1"),
	)

	if value != 1 {
		t.Fatalf(
			"expected 1, got %v",
			value,
		)
	}
}
