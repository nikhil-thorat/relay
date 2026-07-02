package health

import (
	"context"
	"net"
	"time"

	"github.com/nikhil-thorat/relay/internal/logging"
	"github.com/nikhil-thorat/relay/internal/metrics"
	"github.com/nikhil-thorat/relay/internal/target"
)

type Checker struct {
	pool     *target.Pool
	metrics  *metrics.Metrics
	interval time.Duration
	timeout  time.Duration
	logger   *logging.Logger
}

func New(pool *target.Pool, metrics *metrics.Metrics, interval time.Duration, timeout time.Duration, logger *logging.Logger) *Checker {
	return &Checker{
		pool:     pool,
		metrics:  metrics,
		interval: interval,
		timeout:  timeout,
		logger:   logger,
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

		oldState, err := checker.pool.GetState(target.ID)
		if err != nil {
			continue
		}

		healthy := checker.Check(target)
		_ = checker.pool.SetHealthy(target.ID, healthy)

		newState, err := checker.pool.GetState(target.ID)
		if err != nil {
			continue
		}

		if !oldState.Healthy && newState.Healthy {
			checker.logger.Info("target recovered", "target", target.ID)
		}

		if oldState.Healthy && !newState.Healthy {
			checker.logger.Warn("target became unhealthy", "target", target.ID)
		}
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
