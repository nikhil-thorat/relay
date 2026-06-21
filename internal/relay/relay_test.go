package relay

import (
	"testing"

	"github.com/nikhil-thorat/relay/internal/config"
)

func TestNew(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		cfg := &config.Config{
			Strategy: config.StrategyConfig{
				Type: "round_robin",
			},
			Targets: []config.TargetConfig{
				{
					ID:      "api_1",
					Address: "localhost:9001",
					Weight:  1,
				},
			},
		}

		relay, err := New(cfg)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if relay == nil {
			t.Fatal("expected relay, got nil")
		}
	})

	t.Run("invalid strategy", func(t *testing.T) {
		cfg := &config.Config{
			Strategy: config.StrategyConfig{
				Type: "unknown",
			},
		}

		_, err := New(cfg)

		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("targets loaded into pool", func(t *testing.T) {
		cfg := &config.Config{
			Strategy: config.StrategyConfig{
				Type: "round_robin",
			},
			Targets: []config.TargetConfig{
				{
					ID:      "api_1",
					Address: "localhost:9001",
					Weight:  1,
				},
				{
					ID:      "api_2",
					Address: "localhost:9002",
					Weight:  1,
				},
			},
		}

		relay, err := New(cfg)
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

	relay, err := New(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if relay == nil {
		t.Fatal("expected relay, got nil")
	}

	if relay.Balancer == nil {
		t.Fatal("expected balancer, got nil")
	}
}
