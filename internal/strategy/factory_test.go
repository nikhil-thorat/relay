package strategy

import "testing"

func TestNew(t *testing.T) {
	tests := []struct {
		name      string
		strategy  string
		expectErr bool
	}{
		{
			name:      "round robin",
			strategy:  "round_robin",
			expectErr: false,
		},
		{
			name:      "unknown strategy",
			strategy:  "unknown",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy, err := New(tt.strategy)

			if tt.expectErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if strategy == nil {
				t.Fatal("expected strategy, got nil")
			}
		})
	}
}
