package health

import (
	"net"
	"time"

	"github.com/nikhil-thorat/relay/internal/target"
)

type Checker struct {
	pool     *target.Pool
	interval time.Duration
	timeout  time.Duration
}

func New(pool *target.Pool, interval time.Duration, timeout time.Duration) *Checker {
	return &Checker{
		pool:     pool,
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
}

func (checker *Checker) Start() {
	checker.Run()

	ticker := time.NewTicker(
		checker.interval,
	)

	go func() {
		for range ticker.C {
			checker.Run()
		}
	}()
}
