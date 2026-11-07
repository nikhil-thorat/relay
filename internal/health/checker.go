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
	client   *http.Client
	wg       sync.WaitGroup
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
		client: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func (checker *Checker) Check(
	target target.Target,
) bool {
	return checker.check(context.Background(), target)
}

func (checker *Checker) check(ctx context.Context, target target.Target) bool {
	if checker.mode == "http" {
		return checker.checkHTTP(ctx, target)
	}

	return checker.checkTCP(ctx, target)
}

func (checker *Checker) checkTCP(ctx context.Context, t target.Target) bool {
	dialer := net.Dialer{Timeout: checker.timeout}
	conn, err := dialer.DialContext(ctx, "tcp", t.Address)

	if err != nil {
		return false
	}

	defer conn.Close()

	return true
}

func (checker *Checker) checkHTTP(ctx context.Context, t target.Target) bool {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		"http://"+t.Address+checker.path,
		nil,
	)
	if err != nil {
		return false
	}

	resp, err := checker.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

func (checker *Checker) Run() {
	checker.run(context.Background())
}

func (checker *Checker) run(ctx context.Context) {
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

			healthy := checker.check(ctx, t)
			if !checker.pool.SetHealthyForAddress(t.ID, t.Address, healthy) {
				return
			}

			newState, err := checker.pool.GetState(t.ID)
			if err != nil {
				return
			}

			if checker.logger != nil {
				if !oldState.Healthy && newState.Healthy {
					checker.logger.Info("target recovered", "target", t.ID)
				}

				if oldState.Healthy && !newState.Healthy {
					checker.logger.Warn("target became unhealthy", "target", t.ID)
				}
			}
		}(t)
	}

	wg.Wait()

	if checker.metrics != nil {
		checker.metrics.SetHealthyTargets(checker.pool.HealthyCount())
	}
}

func (checker *Checker) Start(ctx context.Context) {

	checker.wg.Go(func() {

		checker.run(ctx)

		if checker.interval <= 0 {
			return
		}

		ticker := time.NewTicker(checker.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				checker.run(ctx)
			}
		}
	})
}

func (checker *Checker) Wait() {
	checker.wg.Wait()
}
