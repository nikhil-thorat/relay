package balancer

import (
	"testing"

	"github.com/nikhil-thorat/relay/internal/strategy"
	"github.com/nikhil-thorat/relay/internal/target"
)

type benchmarkCase struct {
	name     string
	strategy strategy.Strategy
}

func benchmarkCases() []benchmarkCase {
	return []benchmarkCase{
		{name: "round_robin", strategy: &strategy.RoundRobin{}},
		{name: "weighted_round_robin", strategy: &strategy.WeightedRoundRobin{}},
		{name: "least_conn", strategy: &strategy.LeastConnections{}},
		{name: "least_response_time", strategy: &strategy.LeastResponseTime{}},
		{name: "consistent_hash", strategy: &strategy.ConsistentHash{}},
	}
}

func benchmarkPool(b *testing.B) *target.Pool {
	b.Helper()

	pool := target.NewPool()
	if err := pool.Add(&target.Target{ID: "api_1", Address: "localhost:9001", Weight: 3}); err != nil {
		b.Fatal(err)
	}
	if err := pool.Add(&target.Target{ID: "api_2", Address: "localhost:9002", Weight: 2}); err != nil {
		b.Fatal(err)
	}
	if err := pool.Add(&target.Target{ID: "api_3", Address: "localhost:9003", Weight: 1}); err != nil {
		b.Fatal(err)
	}
	for _, id := range []string{"api_1", "api_2", "api_3"} {
		if err := pool.SetHealthy(id, true); err != nil {
			b.Fatal(err)
		}
	}

	return pool
}

func BenchmarkBalancerNext(b *testing.B) {
	for _, tc := range benchmarkCases() {
		b.Run(tc.name, func(b *testing.B) {
			bal := New(benchmarkPool(b), tc.strategy)
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				_, _ = bal.Next()
			}
		})
	}
}

func BenchmarkBalancerSelectForRequest(b *testing.B) {
	excluded := map[string]struct{}{"api_1": {}}

	for _, tc := range benchmarkCases() {
		b.Run(tc.name, func(b *testing.B) {
			bal := New(benchmarkPool(b), tc.strategy)
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				_, _ = bal.SelectForRequest("user-42", excluded)
			}
		})
	}
}
