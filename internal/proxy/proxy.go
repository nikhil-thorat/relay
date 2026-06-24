package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/nikhil-thorat/relay/internal/balancer"
	"github.com/nikhil-thorat/relay/internal/metrics"
)

type Proxy struct {
	balancer *balancer.Balancer
	metrics  *metrics.Metrics
}

func New(balancer *balancer.Balancer, metrics *metrics.Metrics) *Proxy {
	return &Proxy{
		balancer: balancer,
		metrics:  metrics,
	}
}

func (proxy *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	proxy.metrics.IncrementRequests()

	target, err := proxy.balancer.Next()
	if err != nil {
		http.Error(
			w,
			"no available targets",
			http.StatusBadGateway,
		)
		proxy.metrics.IncrementErrors()
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
		return
	}

	reverseProxy := httputil.NewSingleHostReverseProxy(
		targetUrl,
	)

	reverseProxy.ServeHTTP(w, r)

}
