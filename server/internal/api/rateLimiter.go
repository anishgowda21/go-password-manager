package api

import (
	"math"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"
)

var trustedHops = hopsFromEnv()

// Whole limiter for one route
type rateLimiter struct {
	mu        sync.Mutex
	buckets   map[string]*bucket //Ip to client bucket(per route since one rate limiter per route)
	burst     float64            //Quick session req
	every     time.Duration      //sustained rate req
	lastSweep time.Time
}

// How many token each client has and last time we checked it
type bucket struct {
	tokens float64 //Refil in fractions so float
	last   time.Time
}

// Simple constructor for new ratelimiter
func newRateLimiter(burst int, every time.Duration) *rateLimiter {
	return &rateLimiter{
		buckets:   make(map[string]*bucket),
		burst:     float64(burst),
		every:     every,
		lastSweep: time.Now(),
	}
}

func (l *rateLimiter) allow(key string) (bool, time.Duration) {
	//Mutex lock to ensure no two go routinws can run allow at same time on same rate limiter
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	l.sweep(now)

	//find or create the bucket.
	b, ok := l.buckets[key]

	if !ok {
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[key] = b
	}

	//Token refil: there is no timer basically we calculate how much must the clients
	//bucket must be filled from his last req
	//now.Sub(b.last) :- elapsed time and dividing by every gives no of tokens earned
	b.tokens = math.Min(l.burst, b.tokens+float64(now.Sub(b.last))/float64(l.every))
	b.last = now

	//spend or deny
	if b.tokens > 1 {
		b.tokens--
		return true, 0
	}

	return false, time.Duration((1 - b.tokens) * float64(l.every))
}

// Delets bucked that have been idle long enough to have fully refilled.
func (l *rateLimiter) sweep(now time.Time) {
	if now.Sub(l.lastSweep) < time.Minute {
		return
	}

	l.lastSweep = now
	idle := time.Duration(l.burst) * l.every
	for k, b := range l.buckets {
		if now.Sub(b.last) >= idle {
			delete(l.buckets, k)
		}
	}
}

func clientIP(r *http.Request) string {
	if trustedHops > 0 {
		header := strings.Join(r.Header.Values("X-Forwarded-For"), ",")
		if ip, ok := forwardedIP(header, trustedHops); ok {
			return limiterKey(ip)
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if ip, err := netip.ParseAddr(host); err == nil {
		return limiterKey(ip)
	}
	return host
}

// forwardedIP counts hops from the RIGHT of X-Forwarded-For, because the
// proxies we trust append there. Anything further left is client-controlled.
func forwardedIP(header string, hops int) (netip.Addr, bool) {
	parts := strings.Split(header, ",")
	if len(parts) < hops {
		return netip.Addr{}, false
	}
	ip, err := netip.ParseAddr(strings.TrimSpace(parts[len(parts)-hops]))
	if err != nil {
		return netip.Addr{}, false
	}
	return ip, true
}

// limiterKey turns an address into the bucket key. IPv6 clients are grouped
// by /64, since one connection usually owns a whole /64 and could otherwise
// rotate addresses to get a fresh bucket every request.
func limiterKey(ip netip.Addr) string {
	ip = ip.Unmap().WithZone("")
	if ip.Is6() {
		p, _ := ip.Prefix(64)
		return p.String()
	}
	return ip.String()
}

func rateLimit(l *rateLimiter, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if ok, wait := l.allow(clientIP(r)); !ok {
			w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(wait.Seconds()))))
			writeError(w, http.StatusTooManyRequests, "too many requests")
			return
		}
		next(w, r)
	}
}
