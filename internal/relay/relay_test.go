package relay

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikhil-thorat/relay/internal/config"
)

func testRegistry() *prometheus.Registry {
	return prometheus.NewRegistry()
}

func TestNew(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		cfg := &config.Config{
			Server: config.ServerConfig{
				Address: ":8080",
			},
			Strategy: config.StrategyConfig{
				Type: "round_robin",
			},
			Health: config.HealthConfig{
				Enabled: true,
			},
			Metrics: config.MetricsConfig{
				Enabled: true,
				Address: ":9090",
			},
			Targets: []config.TargetConfig{
				{
					ID:      "api_1",
					Address: "localhost:9001",
					Weight:  1,
				},
			},
		}

		relay, err := New(
			cfg,
			testRegistry(),
		)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if relay == nil {
			t.Fatal("expected relay, got nil")
		}

		if relay.Balancer == nil {
			t.Fatal("expected balancer, got nil")
		}

		if relay.Health == nil {
			t.Fatal("expected health checker, got nil")
		}

		if relay.Metrics == nil {
			t.Fatal("expected metrics, got nil")
		}

		if relay.proxy == nil {
			t.Fatal("expected proxy, got nil")
		}

		if relay.server == nil {
			t.Fatal("expected http server, got nil")
		}

		if relay.metricsServer == nil {
			t.Fatal("expected metrics server, got nil")
		}

		if !relay.healthEnabled {
			t.Fatal("expected health to be enabled")
		}

		if !relay.metricsEnabled {
			t.Fatal("expected metrics to be enabled")
		}
	})

	t.Run("invalid strategy", func(t *testing.T) {
		cfg := &config.Config{
			Strategy: config.StrategyConfig{
				Type: "unknown",
			},
		}

		_, err := New(
			cfg,
			testRegistry(),
		)

		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("targets loaded into pool", func(t *testing.T) {
		cfg := &config.Config{
			Server: config.ServerConfig{
				Address: ":8080",
			},
			Strategy: config.StrategyConfig{
				Type: "round_robin",
			},
			Targets: []config.TargetConfig{
				{
					ID:      "api_1",
					Address: "localhost:9001",
				},
				{
					ID:      "api_2",
					Address: "localhost:9002",
				},
			},
		}

		relay, err := New(
			cfg,
			testRegistry(),
		)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if relay.Balancer == nil {
			t.Fatal("expected balancer")
		}
	})
}

func TestNewFromConfig(t *testing.T) {
	cfg, err := config.Load(
		"../../examples/config/explicit.yml",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	relay, err := New(
		cfg,
		testRegistry(),
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if relay == nil {
		t.Fatal("expected relay, got nil")
	}

	if relay.Balancer == nil {
		t.Fatal("expected balancer, got nil")
	}

	if relay.Health == nil {
		t.Fatal("expected health checker, got nil")
	}

	if relay.Metrics == nil {
		t.Fatal("expected metrics, got nil")
	}

	if relay.proxy == nil {
		t.Fatal("expected proxy, got nil")
	}

	if relay.server == nil {
		t.Fatal("expected http server, got nil")
	}

	if relay.metricsServer == nil {
		t.Fatal("expected metrics server, got nil")
	}
}

func TestHealthDisabledMarksTargetsHealthy(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Address: ":8080"},
		Strategy: config.StrategyConfig{Type: "round_robin"},
		Targets: []config.TargetConfig{
			{ID: "api_1", Address: "localhost:9001"},
			{ID: "api_2", Address: "localhost:9002"},
		},
		Health:  config.HealthConfig{Enabled: false},
		Metrics: config.MetricsConfig{Enabled: false},
		Logging: config.LoggingConfig{Level: "off"},
	}

	r, err := New(cfg, testRegistry())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(r.pool.Healthy()) != 2 {
		t.Fatalf("expected 2 healthy targets, got %d", len(r.pool.Healthy()))
	}
}

func TestHealthDisabledEnablesPassiveRecovery(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Address: ":8080"},
		Strategy: config.StrategyConfig{Type: "round_robin"},
		Targets:  []config.TargetConfig{{ID: "api_1", Address: "localhost:9001"}},
		Health:   config.HealthConfig{Enabled: false},
		Metrics:  config.MetricsConfig{Enabled: false},
		Logging:  config.LoggingConfig{Level: "off"},
	}

	r, err := New(cfg, testRegistry())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for range 3 {
		r.Balancer.ReportFailure("api_1")
	}

	state, _ := r.pool.GetState("api_1")
	if state.Healthy {
		t.Fatal("expected target to become unhealthy")
	}

	r.Balancer.ReportSuccess("api_1")
	state, _ = r.pool.GetState("api_1")
	if !state.Healthy {
		t.Fatal("expected passive recovery")
	}
}
