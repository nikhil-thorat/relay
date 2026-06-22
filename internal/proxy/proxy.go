package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/nikhil-thorat/relay/internal/balancer"
)

type Proxy struct {
	balancer *balancer.Balancer
}

func New(balancer *balancer.Balancer) *Proxy {
	return &Proxy{
		balancer: balancer,
	}
}

func (proxy *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	target, err := proxy.balancer.Next()
	if err != nil {
		http.Error(
			w,
			"no available targets",
			http.StatusBadGateway,
		)
		return
	}

	targetUrl, err := url.Parse("http://" + target.Address)
	if err != nil {
		http.Error(
			w,
			"invalid target url",
			http.StatusBadGateway,
		)
		return
	}

	reverseProxy := httputil.NewSingleHostReverseProxy(
		targetUrl,
	)

	reverseProxy.ServeHTTP(w, r)

}
