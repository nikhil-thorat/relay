package proxy

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/nikhil-thorat/relay/internal/balancer"
	"github.com/nikhil-thorat/relay/internal/logging"
	"github.com/nikhil-thorat/relay/internal/metrics"
)

type Proxy struct {
	balancer      *balancer.Balancer
	metrics       *metrics.Metrics
	logger        *logging.Logger
	transport     *http.Transport
	retryAttempts int
	maxInflight   int
	inflight      chan struct{}
	limiter       *ipLimiter
}

func New(balancer *balancer.Balancer, metrics *metrics.Metrics, logger *logging.Logger, retryAttempts int) *Proxy {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
	}

	if retryAttempts < 0 {
		retryAttempts = 0
	}

	return &Proxy{
		balancer:      balancer,
		metrics:       metrics,
		logger:        logger,
		transport:     transport,
		retryAttempts: retryAttempts,
	}
}

func (proxy *Proxy) SetLimits(maxInflight int, rps int) {
	proxy.maxInflight = maxInflight
	if maxInflight > 0 {
		proxy.inflight = make(chan struct{}, maxInflight)
	} else {
		proxy.inflight = nil
	}
	proxy.limiter = newIPLimiter(rps)
}

func requestID(r *http.Request) string {
	if id := r.Header.Get("X-Request-ID"); id != "" {
		return id
	}

	var b [8]byte
	if _, err := rand.Read(b[:]); err == nil {
		return hex.EncodeToString(b[:])
	}

	return ""
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func containsIP(prior string, ip string) bool {
	if ip == "" {
		return true
	}
	return strings.Contains(prior, ip)
}

func isTimeout(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	return errors.Is(err, context.DeadlineExceeded)
}

func itoa(code int) string {
	return strconv.Itoa(code)
}

func (proxy *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	start := time.Now()

	proxy.metrics.IncrementRequests()

	ip := clientIP(r)

	if proxy.limiter != nil && !proxy.limiter.Allow(ip) {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		proxy.metrics.IncrementErrors()
		return
	}

	if proxy.inflight != nil {
		select {
		case proxy.inflight <- struct{}{}:
			defer func() { <-proxy.inflight }()
		default:
			http.Error(w, "too many requests", http.StatusTooManyRequests)
			proxy.metrics.IncrementErrors()
			return
		}
	}

	id := requestID(r)
	if id != "" {
		r.Header.Set("X-Request-ID", id)
	}

	if prior := r.Header.Get("X-Forwarded-For"); prior != "" {
		if !containsIP(prior, ip) {
			r.Header.Set("X-Forwarded-For", prior+", "+ip)
		}
	} else if ip != "" {
		r.Header.Set("X-Forwarded-For", ip)
	}

	attempts := 1
	if (r.Method == http.MethodGet || r.Method == http.MethodHead) && proxy.retryAttempts > 0 {
		attempts = proxy.retryAttempts + 1
	}

	for i := 0; i < attempts; i++ {
		target, err := proxy.balancer.NextForRequest(r)
		if err != nil {
			http.Error(
				w,
				"no available targets",
				http.StatusServiceUnavailable,
			)
			proxy.metrics.IncrementErrors()
			proxy.metrics.ObserveRequestDuration(time.Since(start).Seconds())
			proxy.logger.Error("no healthy targets available", "request-id", id, "method", r.Method, "path", r.URL.Path)

			return
		}

		proxy.balancer.Before(target)
		attemptStart := time.Now()

		targetURL, err := url.Parse("http://" + target.Address)
		if err != nil {
			http.Error(
				w,
				"invalid target url",
				http.StatusBadGateway,
			)
			proxy.metrics.IncrementErrors()
			proxy.metrics.ObserveRequestDuration(time.Since(start).Seconds())
			proxy.logger.Error("invalid target address", "request-id", id, "target", target.ID, "address", target.Address, "method", r.Method, "path", r.URL.Path, "error", err)
			return
		}

		isLast := i == attempts-1

		var failed bool
		var timeout bool

		reverseProxy := httputil.NewSingleHostReverseProxy(
			targetURL,
		)
		reverseProxy.Transport = proxy.transport
		reverseProxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			failed = true
			timeout = isTimeout(err)
			proxy.balancer.ReportFailure(target.ID)
			if isLast {
				code := http.StatusBadGateway
				if timeout {
					code = http.StatusGatewayTimeout
				}
				proxy.metrics.IncrementErrors()
				proxy.metrics.IncrementTargetRequests(target.ID, itoa(code))
				proxy.metrics.ObserveRequestDuration(time.Since(start).Seconds())
				if timeout {
					proxy.logger.Warn("backend timeout", "request-id", id, "target", target.ID, "method", r.Method, "path", r.URL.Path, "error", err)
					http.Error(w, "backend timeout", http.StatusGatewayTimeout)
					return
				}
				proxy.logger.Error("backend error", "request-id", id, "target", target.ID, "method", r.Method, "path", r.URL.Path, "error", err)
				http.Error(w, "bad gateway", http.StatusBadGateway)
			}
		}

		rw := &responseWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		reverseProxy.ServeHTTP(rw, r)

		attemptDuration := time.Since(attemptStart)
		proxy.balancer.After(target, attemptDuration)

		if failed {
			if !isLast {
				continue
			}
			return
		}

		if rw.status >= 500 {
			proxy.balancer.ReportFailure(target.ID)
			proxy.metrics.IncrementErrors()
		} else {
			proxy.balancer.ReportSuccess(target.ID)
		}

		proxy.metrics.IncrementTargetRequests(target.ID, itoa(rw.status))
		proxy.metrics.ObserveRequestDuration(time.Since(start).Seconds())

		duration := time.Since(start)

		proxy.logger.Info("request completed", "request-id", id, "target", target.ID, "method", r.Method, "path", r.URL.Path, "status", rw.status, "duration", duration)

		return
	}

}
