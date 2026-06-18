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
			name: "no targets",
			cfg: Config{
				Strategy: StrategyConfig{
					Type: "round_robin",
				},
			},
			wantErr: true,
		},
		{
			name: "missing target id",
			cfg: Config{
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
