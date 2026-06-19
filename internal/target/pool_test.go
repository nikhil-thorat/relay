package target

import "testing"

func TestNewPool(t *testing.T) {
	pool := NewPool()

	if pool == nil {
		t.Fatal("expected pool, got nil")
	}

	if pool.Targets == nil {
		t.Fatal("targets map not initialized")
	}

	if pool.States == nil {
		t.Fatal("states map not initialized")
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

	if len(pool.Targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(pool.Targets))
	}

	if len(pool.States) != 1 {
		t.Fatalf("expected 1 state, got %d", len(pool.States))
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

	state, ok := pool.States["api_1"]

	if !ok {
		t.Fatal("expected state to be created")
	}

	if state == nil {
		t.Fatal("expected state, got nil")
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

	if state == nil {
		t.Fatal("expected state, got nil")
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
