package metrics

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	RequestsTotal      prometheus.Counter
	RequestErrorsTotal prometheus.Counter
	HealthyTargets     prometheus.Gauge
	TargetRequests     *prometheus.CounterVec
}

func New(registry prometheus.Registerer) *Metrics {
	requestsTotal := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "relay_requests_total",
			Help: "Total number of requests handled by Relay",
		},
	)

	requestErrorsTotal := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "relay_request_errors_total",
			Help: "Total number of failed requests",
		},
	)

	healthyTargets := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "relay_healthy_targets",
			Help: "Current number of healthy targets",
		},
	)

	targetRequests := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "relay_target_requests_total",
			Help: "Total requests routed to targets",
		},
		[]string{"target"},
	)

	registry.MustRegister(
		requestsTotal,
		requestErrorsTotal,
		healthyTargets,
		targetRequests,
	)

	return &Metrics{
		RequestsTotal:      requestsTotal,
		RequestErrorsTotal: requestErrorsTotal,
		HealthyTargets:     healthyTargets,
		TargetRequests:     targetRequests,
	}
}

func (metrics *Metrics) IncrementRequests() {
	metrics.RequestsTotal.Inc()
}

func (metrics *Metrics) IncrementErrors() {
	metrics.RequestErrorsTotal.Inc()
}

func (metrics *Metrics) SetHealthyTargets(count int) {
	metrics.HealthyTargets.Set(float64(count))
}

func (metrics *Metrics) IncrementTargetRequests(targetID string) {
	metrics.TargetRequests.WithLabelValues(targetID).Inc()
}
