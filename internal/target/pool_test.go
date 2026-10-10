package target

import (
	"sync"
	"testing"
)

func TestNewPool(t *testing.T) {
	pool := NewPool()

	if pool == nil {
		t.Fatal("expected pool, got nil")
	}

	if len(pool.List()) != 0 {
		t.Fatal("expected empty pool")
	}

	if len(pool.Healthy()) != 0 {
		t.Fatal("expected no healthy targets")
	}
}

func TestAdd(t *testing.T) {
	pool := NewPool()

	target := &Target{
		ID:      "api_1",
		Address: "localhost:9001",
		Weight:  1,
	}

	if err := pool.Add(target); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(pool.List()) != 1 {
		t.Fatalf("expected 1 target, got %d", len(pool.List()))
	}

	if _, err := pool.GetState("api_1"); err != nil {
		t.Fatalf("expected state, got error: %v", err)
	}

}

func TestAddDuplicate(t *testing.T) {
	pool := NewPool()

	target := &Target{
		ID:      "api_1",
		Address: "localhost:9001",
		Weight:  1,
	}

	_ = pool.Add(target)

	if err := pool.Add(target); err == nil {
		t.Fatal("expected duplicate target error")
	}

}

func TestGet(t *testing.T) {
	pool := NewPool()

	target := &Target{
		ID:      "api_1",
		Address: "localhost:9001",
		Weight:  1,
	}

	_ = pool.Add(target)

	got, err := pool.Get("api_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.ID != target.ID {
		t.Fatalf("expected %s, got %s", target.ID, got.ID)
	}

}

func TestGetMissing(t *testing.T) {
	pool := NewPool()

	_, err := pool.Get("missing")

	if err == nil {
		t.Fatal("expected error")
	}

}

func TestList(t *testing.T) {
	pool := NewPool()

	_ = pool.Add(&Target{
		ID:      "api_1",
		Address: "localhost:9001",
		Weight:  1,
	})

	_ = pool.Add(&Target{
		ID:      "api_2",
		Address: "localhost:9002",
		Weight:  1,
	})

	targets := pool.List()

	if len(targets) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(targets))
	}

}

func TestStateCreatedOnAdd(t *testing.T) {
	pool := NewPool()

	_ = pool.Add(&Target{
		ID:      "api_1",
		Address: "localhost:9001",
		Weight:  1,
	})

	state, err := pool.GetState("api_1")

	if err != nil {
		t.Fatal("expected state to be created")
	}

	if state.Healthy {
		t.Fatal("expected new target to be unhealthy")
	}

}

func TestGetState(t *testing.T) {
	pool := NewPool()

	_ = pool.Add(&Target{
		ID:      "api_1",
		Address: "localhost:9001",
		Weight:  1,
	})

	state, err := pool.GetState("api_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if state.Healthy {
		t.Fatal("expected new target to be unhealthy")
	}

}

func TestGetStateMissing(t *testing.T) {
	pool := NewPool()

	_, err := pool.GetState("missing")

	if err == nil {
		t.Fatal("expected error")
	}

}

func TestSetHealthyTrue(t *testing.T) {
	pool := NewPool()

	_ = pool.Add(&Target{
		ID:      "api_1",
		Address: "localhost:9001",
		Weight:  1,
	})

	if err := pool.SetHealthy("api_1", true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	state, _ := pool.GetState("api_1")

	if !state.Healthy {
		t.Fatal("expected target to be healthy")
	}

}

func TestSetHealthyFalse(t *testing.T) {
	pool := NewPool()

	_ = pool.Add(&Target{
		ID:      "api_1",
		Address: "localhost:9001",
		Weight:  1,
	})

	_ = pool.SetHealthy("api_1", true)
	_ = pool.SetHealthy("api_1", false)

	state, _ := pool.GetState("api_1")

	if state.Healthy {
		t.Fatal("expected target to be unhealthy")
	}

}

func TestHealthy(t *testing.T) {
	pool := NewPool()

	_ = pool.Add(&Target{
		ID: "api_1",
	})

	_ = pool.Add(&Target{
		ID: "api_2",
	})

	_ = pool.SetHealthy("api_1", true)

	targets := pool.Healthy()

	if len(targets) != 1 {
		t.Fatalf("expected 1 healthy target, got %d", len(targets))
	}

	if targets[0].ID != "api_1" {
		t.Fatalf("expected api_1, got %s", targets[0].ID)
	}

}

func TestHealthyEmpty(t *testing.T) {
	pool := NewPool()

	_ = pool.Add(&Target{
		ID: "api_1",
	})

	targets := pool.Healthy()

	if len(targets) != 0 {
		t.Fatalf("expected 0 healthy targets, got %d", len(targets))
	}

}

func TestListPreservesOrder(t *testing.T) {
	pool := NewPool()

	_ = pool.Add(&Target{
		ID: "api_1",
	})

	_ = pool.Add(&Target{
		ID: "api_2",
	})

	_ = pool.Add(&Target{
		ID: "api_3",
	})

	targets := pool.List()

	expected := []string{
		"api_1",
		"api_2",
		"api_3",
	}

	if len(targets) != len(expected) {
		t.Fatalf(
			"expected %d targets, got %d",
			len(expected),
			len(targets),
		)
	}

	for i, want := range expected {
		if targets[i].ID != want {
			t.Fatalf(
				"expected %s at index %d, got %s",
				want,
				i,
				targets[i].ID,
			)
		}
	}
}

func TestAddNil(t *testing.T) {
	pool := NewPool()

	if err := pool.Add(nil); err == nil {
		t.Fatal("expected error for nil target")
	}
}

func TestGetReturnsCopy(t *testing.T) {
	pool := NewPool()

	_ = pool.Add(&Target{
		ID:      "api_1",
		Address: "localhost:9001",
		Weight:  1,
	})

	got, err := pool.Get("api_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got.Address = "mutated"

	again, err := pool.Get("api_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if again.Address != "localhost:9001" {
		t.Fatalf("expected copy, internal mutated to %q", again.Address)
	}
}

func TestListReturnsCopies(t *testing.T) {
	pool := NewPool()

	_ = pool.Add(&Target{ID: "api_1", Address: "localhost:9001"})

	list := pool.List()
	list[0].Address = "mutated"

	again, _ := pool.Get("api_1")
	if again.Address != "localhost:9001" {
		t.Fatalf("expected copy, got %q", again.Address)
	}
}

func TestRemove(t *testing.T) {
	pool := NewPool()

	_ = pool.Add(&Target{ID: "api_1"})
	_ = pool.Add(&Target{ID: "api_2"})

	if err := pool.Remove("api_1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(pool.List()) != 1 {
		t.Fatalf("expected 1 target, got %d", len(pool.List()))
	}

	if _, err := pool.Get("api_1"); err == nil {
		t.Fatal("expected error for removed target")
	}

	if err := pool.Remove("missing"); err == nil {
		t.Fatal("expected error for missing target")
	}
}

func TestConcurrentAddGetHealthy(t *testing.T) {
	pool := NewPool()

	var wg sync.WaitGroup

	for i := range 10 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := "api_" + string(rune('0'+i))
			_ = pool.Add(&Target{ID: id, Address: "localhost:9001"})
			_, _ = pool.Get(id)
			_ = pool.SetHealthy(id, true)
			_ = pool.List()
			_ = pool.Healthy()
		}(i)
	}

	wg.Wait()

	if len(pool.Healthy()) != 10 {
		t.Fatalf("expected 10 healthy, got %d", len(pool.Healthy()))
	}
}

func TestFailureCounter(t *testing.T) {
	pool := NewPool()
	_ = pool.Add(&Target{ID: "api_1", Address: "localhost:9001"})
	_ = pool.SetHealthy("api_1", true)

	_ = pool.RecordFailure("api_1")
	_ = pool.RecordFailure("api_1")

	state, _ := pool.GetState("api_1")
	if state.ConsecutiveFailures != 2 {
		t.Fatalf("expected 2 failures, got %d", state.ConsecutiveFailures)
	}
	if !state.Healthy {
		t.Fatal("expected still healthy before threshold")
	}

	_ = pool.RecordFailure("api_1")
	state, _ = pool.GetState("api_1")
	if state.Healthy {
		t.Fatal("expected unhealthy after 3 failures")
	}

	_ = pool.RecordSuccess("api_1")
	state, _ = pool.GetState("api_1")
	if state.ConsecutiveFailures != 0 {
		t.Fatalf("expected reset, got %d", state.ConsecutiveFailures)
	}
}
