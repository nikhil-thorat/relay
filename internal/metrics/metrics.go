package metrics

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	RequestsTotal      prometheus.Counter
	RequestErrorsTotal prometheus.Counter
	HealthyTargets     prometheus.Gauge
	TargetRequests     *prometheus.CounterVec
	RequestDuration    prometheus.Histogram
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
		[]string{"target", "code"},
	)

	requestDuration := prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "relay_request_duration_seconds",
			Help:    "Request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	registry.MustRegister(
		requestsTotal,
		requestErrorsTotal,
		healthyTargets,
		targetRequests,
		requestDuration,
	)

	return &Metrics{
		RequestsTotal:      requestsTotal,
		RequestErrorsTotal: requestErrorsTotal,
		HealthyTargets:     healthyTargets,
		TargetRequests:     targetRequests,
		RequestDuration:    requestDuration,
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

func (metrics *Metrics) IncrementTargetRequests(targetID string, code string) {
	metrics.TargetRequests.WithLabelValues(targetID, code).Inc()
}

func (metrics *Metrics) ObserveRequestDuration(seconds float64) {
	metrics.RequestDuration.Observe(seconds)
}
