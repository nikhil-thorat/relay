package balancer

import (
	"testing"

	"github.com/nikhil-thorat/relay/internal/strategy"
	"github.com/nikhil-thorat/relay/internal/target"
)

func BenchmarkBalancerNext(b *testing.B) {
	pool := target.NewPool()
	_ = pool.Add(&target.Target{ID: "api_1", Address: "localhost:9001"})
	_ = pool.Add(&target.Target{ID: "api_2", Address: "localhost:9002"})
	_ = pool.SetHealthy("api_1", true)
	_ = pool.SetHealthy("api_2", true)

	bal := New(pool, &strategy.RoundRobin{})

	for b.Loop() {
		_, _ = bal.Next()
	}
}
