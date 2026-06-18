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
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Load(test.path)

			if test.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}

			if !test.wantErr && err != nil {
				t.Fatalf("expected nil, got %v", err)
			}
		})
	}
}
