package balancer

import (
	"net/http"
	"time"

	"github.com/nikhil-thorat/relay/internal/strategy"
	"github.com/nikhil-thorat/relay/internal/target"
)

type Balancer struct {
	pool     *target.Pool
	strategy strategy.Strategy
}

func New(pool *target.Pool, strategy strategy.Strategy) *Balancer {
	return &Balancer{
		pool:     pool,
		strategy: strategy,
	}
}

func (balancer *Balancer) SetStrategy(s strategy.Strategy) {
	balancer.strategy = s
}

func (balancer *Balancer) Next() (target.Target, error) {
	return balancer.strategy.Select(
		balancer.pool.Healthy(),
	)
}

func (balancer *Balancer) NextForRequest(r *http.Request) (target.Target, error) {
	healthy := balancer.pool.Healthy()

	if keyed, ok := balancer.strategy.(interface {
		SelectWithKey([]target.Target, string) (target.Target, error)
	}); ok {
		key := r.Header.Get("X-Request-ID")
		if key == "" {
			key = r.RemoteAddr
		}
		return keyed.SelectWithKey(healthy, key)
	}

	return balancer.strategy.Select(healthy)
}

func (balancer *Balancer) Before(t target.Target) {
	if tracker, ok := balancer.strategy.(interface{ Inc(string) }); ok {
		tracker.Inc(t.ID)
	}
}

func (balancer *Balancer) After(t target.Target, d time.Duration) {
	if tracker, ok := balancer.strategy.(interface{ Done(string) }); ok {
		tracker.Done(t.ID)
	}
	if tracker, ok := balancer.strategy.(interface {
		Observe(string, time.Duration)
	}); ok {
		tracker.Observe(t.ID, d)
	}
}

func (balancer *Balancer) ReportFailure(ID string) {
	_ = balancer.pool.RecordFailure(ID)
}

func (balancer *Balancer) ReportSuccess(ID string) {
	_ = balancer.pool.RecordSuccess(ID)
}
