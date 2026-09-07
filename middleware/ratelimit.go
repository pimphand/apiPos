package middleware

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// rateLimitRule is a per-window limit for one RPC (or a group, when method is
// an exact match or a prefix).
type rateLimitRule struct {
	// method is the full gRPC method path this rule applies to. When it ends
	// with "/*" it matches every method under that service prefix.
	method string
	// limit is the max number of calls allowed per window.
	limit int
	// window is the sliding window duration.
	window time.Duration
}

// selfOrderRateLimits guards the public self-order endpoints against abuse
// (docs/pos-self-order.md §5.5): generous read limits, strict write limits.
var selfOrderRateLimits = []rateLimitRule{
	{method: "/selforder.v1.SelfOrderService/CreatePublicOrder", limit: 10, window: time.Minute},
	{method: "/selforder.v1.SelfOrderService/PayPublicOrder", limit: 60, window: time.Minute},
	{method: "/selforder.v1.SelfOrderService/*", limit: 120, window: time.Minute},
}

// windowEntry is one observed call inside a sliding window.
type windowEntry struct {
	at time.Time
}

// ipLimiter tracks per-IP call timestamps for a single rule.
type ipLimiter struct {
	mu      sync.Mutex
	entries map[string][]windowEntry
}

func (l *ipLimiter) allow(ip string, rule rateLimitRule) bool {
	now := time.Now()
	cutoff := now.Add(-rule.window)

	l.mu.Lock()
	defer l.mu.Unlock()

	queue := l.entries[ip]
	kept := queue[:0]
	for _, e := range queue {
		if e.at.After(cutoff) {
			kept = append(kept, e)
		}
	}
	if len(kept) >= rule.limit {
		l.entries[ip] = kept
		return false
	}
	kept = append(kept, windowEntry{at: now})
	l.entries[ip] = kept
	return true
}

// rateLimiter is a set of per-rule limiters keyed by client IP.
type rateLimiter struct {
	mu    sync.Mutex
	rules []rateLimitRule
	per   map[int]*ipLimiter
}

func newRateLimiter(rules []rateLimitRule) *rateLimiter {
	per := make(map[int]*ipLimiter, len(rules))
	for i := range rules {
		per[i] = &ipLimiter{entries: map[string][]windowEntry{}}
	}
	return &rateLimiter{rules: rules, per: per}
}

// allow checks the matching rules for the IP (most specific rule wins).
func (r *rateLimiter) allow(ip, method string) bool {
	r.mu.Lock()
	rules := append([]rateLimitRule(nil), r.rules...)
	r.mu.Unlock()

	best := -1
	for i, rule := range rules {
		if rule.method == method || (strings.HasSuffix(rule.method, "/*") && strings.HasPrefix(method, strings.TrimSuffix(rule.method, "*"))) {
			if best == -1 || ruleMethodSpecificity(rule) > ruleMethodSpecificity(rules[best]) {
				best = i
			}
		}
	}
	if best == -1 {
		return true
	}
	return r.per[best].allow(ip, rules[best])
}

// ruleMethodSpecificity ranks exact matches above service-wide wildcards so
// the strict per-method limits (submit/pay) win over the generic limit.
func ruleMethodSpecificity(r rateLimitRule) int {
	if strings.HasSuffix(r.method, "/*") {
		return 1
	}
	return 2
}

// clientIP extracts the caller's IP, preferring the first x-forwarded-for
// value (set by the grpc-gateway / reverse proxy) and falling back to the
// gRPC peer address.
func clientIP(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if v := md.Get("x-forwarded-for"); len(v) > 0 {
			first := v[0]
			if idx := strings.IndexByte(first, ','); idx >= 0 {
				first = first[:idx]
			}
			if ip := net.ParseIP(strings.TrimSpace(first)); ip != nil {
				return ip.String()
			}
		}
	}
	if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
		if host, _, err := net.SplitHostPort(p.Addr.String()); err == nil {
			return host
		}
		return p.Addr.String()
	}
	return "unknown"
}

// UnaryServerRateLimitInterceptor returns an interceptor that rate limits the
// configured public methods per client IP. Unlisted methods pass through.
func UnaryServerRateLimitInterceptor() grpc.UnaryServerInterceptor {
	limiter := newRateLimiter(selfOrderRateLimits)
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		ip := clientIP(ctx)
		if !limiter.allow(ip, info.FullMethod) {
			return nil, status.Error(codes.ResourceExhausted, "Too many requests, please try again later")
		}
		return handler(ctx, req)
	}
}