package proxy

import (
	"sync"
	"time"
)

const (
	bucketTTL  = 10 * time.Minute
	sweepEvery = 1024
)

type tokenBucket struct {
	tokens float64
	last   time.Time
}

type ipLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*tokenBucket
	rps      float64
	burst    float64
	requests int
}

func newIPLimiter(rps int) *ipLimiter {
	if rps <= 0 {
		return nil
	}
	return &ipLimiter{
		buckets: make(map[string]*tokenBucket),
		rps:     float64(rps),
		burst:   float64(rps),
	}
}

func (l *ipLimiter) Allow(ip string) bool {
	if l == nil {
		return true
	}

	now := time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	b, ok := l.buckets[ip]
	if !ok {
		b = &tokenBucket{tokens: l.burst, last: now}
		l.buckets[ip] = b
	}

	elapsed := now.Sub(b.last).Seconds()
	b.tokens += elapsed * l.rps
	if b.tokens > l.burst {
		b.tokens = l.burst
	}
	b.last = now

	l.requests++
	if l.requests%sweepEvery == 0 {
		for key, bucket := range l.buckets {
			if now.Sub(bucket.last) > bucketTTL {
				delete(l.buckets, key)
			}
		}
	}

	if b.tokens < 1 {
		return false
	}

	b.tokens--
	return true
}
