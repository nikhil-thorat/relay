package config

import "testing"

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: Config{
				Server: ServerConfig{
					Address: ":42069",
				},
				Strategy: StrategyConfig{
					Type: "round_robin",
				},
				Targets: []TargetConfig{
					{
						ID:      "api_1",
						Address: "localhost:9001",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid weighted strategy",
			cfg: Config{
				Server:   ServerConfig{Address: ":42069"},
				Strategy: StrategyConfig{Type: "weighted_round_robin"},
				Targets: []TargetConfig{
					{ID: "api_1", Address: "localhost:9001"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid least_conn strategy",
			cfg: Config{
				Server:   ServerConfig{Address: ":42069"},
				Strategy: StrategyConfig{Type: "least_conn"},
				Targets: []TargetConfig{
					{ID: "api_1", Address: "localhost:9001"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid least_response_time strategy",
			cfg: Config{
				Server:   ServerConfig{Address: ":42069"},
				Strategy: StrategyConfig{Type: "least_response_time"},
				Targets: []TargetConfig{
					{ID: "api_1", Address: "localhost:9001"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid consistent_hash strategy",
			cfg: Config{
				Server:   ServerConfig{Address: ":42069"},
				Strategy: StrategyConfig{Type: "consistent_hash"},
				Targets: []TargetConfig{
					{ID: "api_1", Address: "localhost:9001"},
				},
			},
			wantErr: false,
		},
		{
			name: "no targets",
			cfg: Config{
				Server: ServerConfig{
					Address: ":42069",
				},
				Strategy: StrategyConfig{
					Type: "round_robin",
				},
			},
			wantErr: true,
		},
		{
			name: "missing target id",
			cfg: Config{
				Server: ServerConfig{
					Address: ":42069",
				},
				Strategy: StrategyConfig{
					Type: "round_robin",
				},
				Targets: []TargetConfig{
					{
						Address: "localhost:9001",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "missing target address",
			cfg: Config{
				Server: ServerConfig{
					Address: ":42069",
				},
				Strategy: StrategyConfig{
					Type: "round_robin",
				},
				Targets: []TargetConfig{
					{
						ID: "api_1",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid strategy",
			cfg: Config{
				Server: ServerConfig{
					Address: ":42069",
				},
				Strategy: StrategyConfig{
					Type: "random",
				},
				Targets: []TargetConfig{
					{
						ID:      "api_1",
						Address: "localhost:9001",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "duplicate target ids",
			cfg: Config{
				Server: ServerConfig{
					Address: ":42069",
				},
				Strategy: StrategyConfig{
					Type: "round_robin",
				},
				Targets: []TargetConfig{
					{
						ID:      "api_1",
						Address: "localhost:9001",
					},
					{
						ID:      "api_1",
						Address: "localhost:9002",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "empty server address",
			cfg: Config{
				Server: ServerConfig{
					Address: "",
				},
				Strategy: StrategyConfig{
					Type: "round_robin",
				},
				Targets: []TargetConfig{
					{
						ID:      "api_1",
						Address: "localhost:9001",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "metrics enabled without address",
			cfg: Config{
				Server: ServerConfig{
					Address: ":42069",
				},
				Strategy: StrategyConfig{
					Type: "round_robin",
				},
				Targets: []TargetConfig{
					{
						ID:      "api_1",
						Address: "localhost:9001",
					},
				},
				Metrics: MetricsConfig{
					Enabled: true,
					Address: "",
				},
			},
			wantErr: true,
		},
		{
			name: "health enabled with zero interval",
			cfg: Config{
				Server: ServerConfig{
					Address: ":42069",
				},
				Strategy: StrategyConfig{
					Type: "round_robin",
				},
				Targets: []TargetConfig{
					{
						ID:      "api_1",
						Address: "localhost:9001",
					},
				},
				Health: HealthConfig{
					Enabled:  true,
					Interval: 0,
					Timeout:  1000000000,
				},
			},
			wantErr: true,
		},
		{
			name: "health enabled with zero timeout",
			cfg: Config{
				Server: ServerConfig{
					Address: ":42069",
				},
				Strategy: StrategyConfig{
					Type: "round_robin",
				},
				Targets: []TargetConfig{
					{
						ID:      "api_1",
						Address: "localhost:9001",
					},
				},
				Health: HealthConfig{
					Enabled:  true,
					Interval: 1000000000,
					Timeout:  0,
				},
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.cfg.Validate()

			if test.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}

			if !test.wantErr && err != nil {
				t.Fatalf("expected nil, got %v", err)
			}
		})
	}

}
