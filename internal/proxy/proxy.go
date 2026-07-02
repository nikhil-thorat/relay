package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/nikhil-thorat/relay/internal/balancer"
	"github.com/nikhil-thorat/relay/internal/logging"
	"github.com/nikhil-thorat/relay/internal/metrics"
)

type Proxy struct {
	balancer *balancer.Balancer
	metrics  *metrics.Metrics
	logger   *logging.Logger
}

func New(balancer *balancer.Balancer, metrics *metrics.Metrics, logger *logging.Logger) *Proxy {
	return &Proxy{
		balancer: balancer,
		metrics:  metrics,
		logger:   logger,
	}
}

func (proxy *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	start := time.Now()

	proxy.metrics.IncrementRequests()

	target, err := proxy.balancer.Next()
	if err != nil {
		http.Error(
			w,
			"no available targets",
			http.StatusBadGateway,
		)
		proxy.metrics.IncrementErrors()
		proxy.logger.Error("no healthy targets available", "method", r.Method, "path", r.URL.Path)

		return
	}

	proxy.metrics.IncrementTargetRequests(
		target.ID,
	)

	targetUrl, err := url.Parse("http://" + target.Address)
	if err != nil {
		http.Error(
			w,
			"invalid target url",
			http.StatusBadGateway,
		)
		proxy.metrics.IncrementErrors()
		proxy.logger.Error("invalid target address", "target", "address", target.Address, target.ID, "method", r.Method, "path", r.URL.Path, "error", err)
		return
	}

	reverseProxy := httputil.NewSingleHostReverseProxy(
		targetUrl,
	)

	rw := &responseWriter{
		ResponseWriter: w,
		status:         http.StatusOK,
	}

	reverseProxy.ServeHTTP(rw, r)

	duration := time.Since(start)

	proxy.logger.Info("request completed", "target", target.ID, "method", r.Method, "path", r.URL.Path, "status", rw.status, "duration", duration)

}
