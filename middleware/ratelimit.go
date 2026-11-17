package middleware

import (
	"context"
	"net"
	"os"
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

// authRateLimits throttles the unauthenticated credential endpoints per client
// IP. It is the first of two layers: per-account lockout (internal/auth)
// handles slow, distributed guessing against a single username, while these
// limits stop one host from spraying many usernames or mass-registering stores.
var authRateLimits = []rateLimitRule{
	{method: "/auth.v1.AuthService/Login", limit: 10, window: time.Minute},
	{method: "/auth.v1.AuthService/Register", limit: 5, window: 10 * time.Minute},
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

// loopbackProxyNets are always treated as trusted forwarding proxies: the
// in-process grpc-gateway dials the gRPC server over loopback, so its peer
// address is a loopback IP.
var loopbackProxyNets = func() []*net.IPNet {
	var nets []*net.IPNet
	for _, cidr := range []string{"127.0.0.0/8", "::1/128"} {
		if _, network, err := net.ParseCIDR(cidr); err == nil {
			nets = append(nets, network)
		}
	}
	return nets
}()

// trustedProxyNets returns the networks whose forwarding headers are believed.
// TRUSTED_PROXIES is a comma-separated list of IPs or CIDRs (for example the
// reverse proxy in front of the API); loopback is always included.
func trustedProxyNets() []*net.IPNet {
	nets := append([]*net.IPNet(nil), loopbackProxyNets...)
	for _, entry := range strings.Split(os.Getenv("TRUSTED_PROXIES"), ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if _, network, err := net.ParseCIDR(entry); err == nil {
			nets = append(nets, network)
			continue
		}
		if ip := net.ParseIP(entry); ip != nil {
			bits := 32
			if ip.To4() == nil {
				bits = 128
			}
			nets = append(nets, &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)})
		}
	}
	return nets
}

func isTrustedProxy(ip net.IP) bool {
	for _, network := range trustedProxyNets() {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

// peerIP returns the address of the TCP peer that opened the gRPC connection.
// Forwarding headers are intentionally ignored here: they are caller supplied
// and must not influence the result.
func peerIP(ctx context.Context) string {
	p, ok := peer.FromContext(ctx)
	if !ok || p.Addr == nil {
		return ""
	}
	if host, _, err := net.SplitHostPort(p.Addr.String()); err == nil {
		return host
	}
	return p.Addr.String()
}

// clientIP extracts the caller's IP.
//
// The gRPC peer address is authoritative. x-forwarded-for and x-real-ip are
// only consulted when that peer is a trusted proxy (loopback or
// TRUSTED_PROXIES), and the X-Forwarded-For chain is read from the right: the
// rightmost entry is the one appended by the nearest proxy, while anything to
// its left can be forged by the client to get a fresh rate-limit bucket.
func clientIP(ctx context.Context) string {
	peerAddr := peerIP(ctx)
	if peerAddr == "" {
		return "unknown"
	}
	peerIP := net.ParseIP(peerAddr)
	if peerIP == nil || !isTrustedProxy(peerIP) {
		return peerAddr
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return peerAddr
	}

	var hops []string
	for _, header := range md.Get("x-forwarded-for") {
		hops = append(hops, strings.Split(header, ",")...)
	}
	for i := len(hops) - 1; i >= 0; i-- {
		ip := net.ParseIP(strings.TrimSpace(hops[i]))
		if ip == nil || isTrustedProxy(ip) {
			continue
		}
		return ip.String()
	}

	if v := md.Get("x-real-ip"); len(v) > 0 {
		if ip := net.ParseIP(strings.TrimSpace(v[0])); ip != nil && !isTrustedProxy(ip) {
			return ip.String()
		}
	}

	return peerAddr
}

// UnaryServerRateLimitInterceptor returns an interceptor that rate limits the
// configured public methods per client IP. Unlisted methods pass through.
func UnaryServerRateLimitInterceptor() grpc.UnaryServerInterceptor {
	rules := make([]rateLimitRule, 0, len(selfOrderRateLimits)+len(authRateLimits))
	rules = append(rules, selfOrderRateLimits...)
	rules = append(rules, authRateLimits...)
	limiter := newRateLimiter(rules)
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
