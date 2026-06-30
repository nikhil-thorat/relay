package health

import (
	"context"
	"net"
	"time"

	"github.com/nikhil-thorat/relay/internal/metrics"
	"github.com/nikhil-thorat/relay/internal/target"
)

type Checker struct {
	pool     *target.Pool
	metrics  *metrics.Metrics
	interval time.Duration
	timeout  time.Duration
}

func New(pool *target.Pool, metrics *metrics.Metrics, interval time.Duration, timeout time.Duration) *Checker {
	return &Checker{
		pool:     pool,
		metrics:  metrics,
		interval: interval,
		timeout:  timeout,
	}
}

func (checker *Checker) Check(
	target *target.Target,
) bool {

	conn, err := net.DialTimeout(
		"tcp",
		target.Address,
		checker.timeout,
	)

	if err != nil {
		return false
	}

	defer conn.Close()

	return true
}

func (checker *Checker) Run() {
	for _, target := range checker.pool.List() {
		healthy := checker.Check(target)
		_ = checker.pool.SetHealthy(target.ID, healthy)
	}

	checker.metrics.SetHealthyTargets(len(checker.pool.Healthy()))
}

func (checker *Checker) Start(ctx context.Context) {
	checker.Run()

	go func() {
		ticker := time.NewTicker(checker.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				checker.Run()
			}
		}
	}()
}
