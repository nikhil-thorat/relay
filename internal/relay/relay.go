package relay

import (
	"context"
	"net/http"

	"github.com/nikhil-thorat/relay/internal/balancer"
	"github.com/nikhil-thorat/relay/internal/config"
	"github.com/nikhil-thorat/relay/internal/health"
	"github.com/nikhil-thorat/relay/internal/logging"
	"github.com/nikhil-thorat/relay/internal/metrics"
	"github.com/nikhil-thorat/relay/internal/proxy"
	"github.com/nikhil-thorat/relay/internal/strategy"
	"github.com/nikhil-thorat/relay/internal/target"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Relay struct {
	Balancer *balancer.Balancer
	Health   *health.Checker
	Metrics  *metrics.Metrics
	Logger   *logging.Logger

	metricsEnabled bool
	healthEnabled  bool
	proxy          *proxy.Proxy
	server         Server
	metricsServer  Server

	ctx    context.Context
	cancel context.CancelFunc
}

func New(cfg *config.Config, registry *prometheus.Registry) (*Relay, error) {
	pool := target.NewPool()

	for _, t := range cfg.Targets {
		err := pool.Add(&target.Target{
			ID:      t.ID,
			Address: t.Address,
			Weight:  t.Weight,
		})
		if err != nil {
			return nil, err
		}
	}

	logger, err := logging.New(cfg.Logging.Level)
	if err != nil {
		return nil, err
	}

	strat, err := strategy.New(cfg.Strategy.Type)
	if err != nil {
		return nil, err
	}

	balancer := balancer.New(pool, strat)

	metrics := metrics.New(
		registry,
	)

	checker := health.New(
		pool,
		metrics,
		cfg.Health.Interval,
		cfg.Health.Timeout,
		logger.WithComponent("health"),
	)

	proxy := proxy.New(balancer, metrics, logger.WithComponent("proxy"))

	server := &http.Server{
		Addr:    cfg.Server.Address,
		Handler: proxy,
	}

	metricsServer := &http.Server{
		Addr: cfg.Metrics.Address,
		Handler: promhttp.HandlerFor(
			registry,
			promhttp.HandlerOpts{},
		),
	}

	ctx, cancel := context.WithCancel(
		context.Background(),
	)

	return &Relay{
		Balancer:       balancer,
		Health:         checker,
		Metrics:        metrics,
		Logger:         logger.WithComponent("relay"),
		healthEnabled:  cfg.Health.Enabled,
		metricsEnabled: cfg.Metrics.Enabled,
		proxy:          proxy,
		server:         server,
		metricsServer:  metricsServer,
		ctx:            ctx,
		cancel:         cancel,
	}, nil
}

func (relay *Relay) startHealth() {

	relay.Logger.Info("starting health checker")

	if relay.healthEnabled && relay.Health != nil {
		relay.Health.Start(relay.ctx)
	}
}

func (relay *Relay) startMetrics() {

	relay.Logger.Info("starting metrics server")

	if !relay.metricsEnabled || relay.metricsServer == nil {
		return
	}

	go func() {
		err := relay.metricsServer.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			// TODO: log error
		}
	}()
}

func (relay *Relay) startHTTP() error {

	relay.Logger.Info("starting http server")

	if relay.server == nil {
		return nil
	}

	err := relay.server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

func (relay *Relay) Start() error {

	relay.Logger.Info("starting relay")

	relay.startHealth()
	relay.startMetrics()
	return relay.startHTTP()
}
