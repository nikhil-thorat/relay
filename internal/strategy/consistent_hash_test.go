package strategy

import (
	"sync"
	"testing"

	"github.com/nikhil-thorat/relay/internal/target"
)

func TestConsistentHashEmptyReturnsError(t *testing.T) {
	c := &ConsistentHash{}

	if _, err := c.Select(nil); err == nil {
		t.Fatal("expected error")
	}

	if _, err := c.SelectWithKey(nil, "key"); err == nil {
		t.Fatal("expected error")
	}
}

func TestConsistentHashSticky(t *testing.T) {
	c := &ConsistentHash{}

	targets := []target.Target{{ID: "api_1"}, {ID: "api_2"}, {ID: "api_3"}}

	first, err := c.SelectWithKey(targets, "user-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for range 20 {
		s, err := c.SelectWithKey(targets, "user-123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.ID != first.ID {
			t.Fatalf("expected sticky %s, got %s", first.ID, s.ID)
		}
	}
}

func TestConsistentHashDistribution(t *testing.T) {
	c := &ConsistentHash{}

	targets := []target.Target{{ID: "api_1"}, {ID: "api_2"}}

	seen := map[string]bool{}
	for i := range 100 {
		s, _ := c.SelectWithKey(targets, "key-"+string(rune('a'+i%26))+string(rune('0'+i%10)))
		seen[s.ID] = true
	}

	if len(seen) < 2 {
		t.Fatal("expected distribution across both targets")
	}
}

func TestConsistentHashConcurrentSelect(t *testing.T) {
	c := &ConsistentHash{}

	targets := []target.Target{{ID: "api_1"}, {ID: "api_2"}}

	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			for range 50 {
				if _, err := c.Select(targets); err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if _, err := c.SelectWithKey(targets, "k"); err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
			}
		})
	}
	wg.Wait()
}
