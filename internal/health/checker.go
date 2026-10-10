package health

import (
	"context"
	"net"
	"net/http"
	"sync"
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
	mode     string
	path     string
	logger   *logging.Logger
}

func New(pool *target.Pool, metrics *metrics.Metrics, interval time.Duration, timeout time.Duration, mode string, path string, logger *logging.Logger) *Checker {
	if mode == "" {
		mode = "tcp"
	}
	if path == "" {
		path = "/health"
	}
	return &Checker{
		pool:     pool,
		metrics:  metrics,
		interval: interval,
		timeout:  timeout,
		mode:     mode,
		path:     path,
		logger:   logger,
	}
}

func (checker *Checker) Check(
	target target.Target,
) bool {

	if checker.mode == "http" {
		return checker.checkHTTP(target)
	}

	return checker.checkTCP(target)
}

func (checker *Checker) checkTCP(t target.Target) bool {
	conn, err := net.DialTimeout(
		"tcp",
		t.Address,
		checker.timeout,
	)

	if err != nil {
		return false
	}

	defer conn.Close()

	return true
}

func (checker *Checker) checkHTTP(t target.Target) bool {
	client := &http.Client{
		Timeout: checker.timeout,
	}

	resp, err := client.Get("http://" + t.Address + checker.path)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

func (checker *Checker) Run() {
	targets := checker.pool.List()

	var wg sync.WaitGroup
	sem := make(chan struct{}, 10)

	for _, t := range targets {
		wg.Add(1)
		sem <- struct{}{}
		go func(t target.Target) {
			defer wg.Done()
			defer func() { <-sem }()

			oldState, err := checker.pool.GetState(t.ID)
			if err != nil {
				return
			}

			healthy := checker.Check(t)
			_ = checker.pool.SetHealthy(t.ID, healthy)

			newState, err := checker.pool.GetState(t.ID)
			if err != nil {
				return
			}

			if !oldState.Healthy && newState.Healthy {
				checker.logger.Info("target recovered", "target", t.ID)
			}

			if oldState.Healthy && !newState.Healthy {
				checker.logger.Warn("target became unhealthy", "target", t.ID)
			}
		}(t)
	}

	wg.Wait()

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
