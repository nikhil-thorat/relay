package config

import "testing"

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid config",
			path:    "../../examples/config/explicit.yml",
			wantErr: false,
		},
		{
			name:    "missing config",
			path:    "../../examples/config/does_not_exist.yml",
			wantErr: true,
		},
		{
			name:    "invalid config",
			path:    "../../examples/config/invalid.yml",
			wantErr: true,
		},
		{
			name:    "strategy rr",
			path:    "../../examples/strategies/rr.yml",
			wantErr: false,
		},
		{
			name:    "strategy wrr",
			path:    "../../examples/strategies/wrr.yml",
			wantErr: false,
		},
		{
			name:    "strategy least-conn",
			path:    "../../examples/strategies/least-conn.yml",
			wantErr: false,
		},
		{
			name:    "strategy lrt",
			path:    "../../examples/strategies/lrt.yml",
			wantErr: false,
		},
		{
			name:    "strategy consistent-hash",
			path:    "../../examples/strategies/consistent-hash.yml",
			wantErr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg, err := Load(test.path)

			if test.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}

			if !test.wantErr && err != nil {
				t.Fatalf("expected nil, got %v", err)
			}

			if !test.wantErr && cfg == nil {
				t.Fatal("expected config, got nil")
			}
		})
	}
}

func TestLoadStrategyTypes(t *testing.T) {
	tests := []struct {
		path     string
		strategy string
	}{
		{"../../examples/strategies/rr.yml", "round_robin"},
		{"../../examples/strategies/wrr.yml", "weighted_round_robin"},
		{"../../examples/strategies/least-conn.yml", "least_conn"},
		{"../../examples/strategies/lrt.yml", "least_response_time"},
		{"../../examples/strategies/consistent-hash.yml", "consistent_hash"},
	}

	for _, test := range tests {
		t.Run(test.strategy, func(t *testing.T) {
			cfg, err := Load(test.path)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if cfg.Strategy.Type != test.strategy {
				t.Fatalf("expected %s, got %s", test.strategy, cfg.Strategy.Type)
			}

			if len(cfg.Targets) != 3 {
				t.Fatalf("expected 3 targets, got %d", len(cfg.Targets))
			}
		})
	}
}
