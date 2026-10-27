package api

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/KeKe45/resourcelocate/internal/ratelimit"
)

// limits holds the rate limiters. The numbers are generous for a person
// using the site and tight for a script.
type limits struct {
	trustProxy bool
	// general covers every API request, photos included: bursts of 300,
	// then 5 a second.
	general *ratelimit.Limiter
	// auth covers sign-up, login, demo login and password changes per IP:
	// 10 tries, then one more every 30 seconds.
	auth *ratelimit.Limiter
	// account covers login attempts per email, whatever the IP: 10 tries,
	// then one a minute.
	account *ratelimit.Limiter
}

func newLimits(trustProxy bool) *limits {
	return &limits{
		trustProxy: trustProxy,
		general:    ratelimit.New(300, 200*time.Millisecond),
		auth:       ratelimit.New(10, 30*time.Second),
		account:    ratelimit.New(10, time.Minute),
	}
}

// allow takes a token from l for key, or answers 429 with Retry-After and
// returns false.
func (ls *limits) allow(w http.ResponseWriter, l *ratelimit.Limiter, key string) bool {
	ok, wait := l.Allow(key)
	if ok {
		return true
	}
	secs := int(wait.Seconds()) + 1
	w.Header().Set("Retry-After", fmt.Sprint(secs))
	writeMsg(w, http.StatusTooManyRequests, fmt.Sprintf("Too many attempts. Wait %d seconds and try again.", secs))
	return false
}

// clientIP is the address the request came from. Behind a proxy (Render,
// Fly, a load balancer) the connection comes from the proxy, which puts
// the real client last in X-Forwarded-For; that header is only trusted
// when TRUST_PROXY is set, since anyone can send it.
func (ls *limits) clientIP(r *http.Request) string {
	if ls.trustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			return strings.TrimSpace(parts[len(parts)-1])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// limited applies the general per-IP limit to every request to h.
func (s *Server) limited(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.limits.allow(w, s.limits.general, s.limits.clientIP(r)) {
			h.ServeHTTP(w, r)
		}
	})
}

// authLimited applies the tight per-IP limit for endpoints that check or
// set a password.
func (s *Server) authLimited(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.limits.allow(w, s.limits.auth, s.limits.clientIP(r)) {
			h(w, r)
		}
	}
}
