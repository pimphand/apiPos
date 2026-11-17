package middleware

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

// ctxWithPeer builds a context whose gRPC peer has the given address and that
// carries the supplied forwarding headers.
func ctxWithPeer(peerAddr string, headers map[string]string) context.Context {
	ctx := peer.NewContext(context.Background(), &peer.Peer{
		Addr: &net.TCPAddr{IP: net.ParseIP(peerAddr), Port: 44444},
	})
	if len(headers) == 0 {
		return ctx
	}
	md := metadata.New(nil)
	for key, value := range headers {
		md.Set(key, value)
	}
	return metadata.NewIncomingContext(ctx, md)
}

func TestClientIPIgnoresForwardingHeadersFromUntrustedPeer(t *testing.T) {
	t.Setenv("TRUSTED_PROXIES", "")

	ctx := ctxWithPeer("198.51.100.20", map[string]string{
		"x-forwarded-for": "1.2.3.4, 5.6.7.8",
		"x-real-ip":       "9.9.9.9",
	})

	assert.Equal(t, "198.51.100.20", clientIP(ctx),
		"a direct caller must not be able to pick its own rate-limit bucket")
}

func TestClientIPUsesRightmostForwardedHopFromTrustedProxy(t *testing.T) {
	t.Setenv("TRUSTED_PROXIES", "")

	// The proxy appends the address it saw, so "203.0.113.9" is the real client
	// and the leading "6.6.6.6" is a value the client injected.
	ctx := ctxWithPeer("127.0.0.1", map[string]string{
		"x-forwarded-for": "6.6.6.6, 203.0.113.9",
	})

	assert.Equal(t, "203.0.113.9", clientIP(ctx))
}

func TestClientIPSkipsTrustedHops(t *testing.T) {
	t.Setenv("TRUSTED_PROXIES", "10.0.0.0/8")

	ctx := ctxWithPeer("10.1.2.3", map[string]string{
		"x-forwarded-for": "203.0.113.9, 10.0.0.5, 10.1.2.3",
	})

	assert.Equal(t, "203.0.113.9", clientIP(ctx),
		"trusted hops in the chain are skipped; the first untrusted hop is the client")
}

func TestClientIPFallsBackToPeerWhenChainCarriesNoClient(t *testing.T) {
	t.Setenv("TRUSTED_PROXIES", "")

	ctx := ctxWithPeer("127.0.0.1", map[string]string{
		"x-forwarded-for": "127.0.0.1",
	})

	assert.Equal(t, "127.0.0.1", clientIP(ctx))
}

func TestClientIPIgnoresMalformedForwardedValues(t *testing.T) {
	t.Setenv("TRUSTED_PROXIES", "")

	ctx := ctxWithPeer("127.0.0.1", map[string]string{
		"x-forwarded-for": "not-an-ip, 203.0.113.9",
	})

	assert.Equal(t, "203.0.113.9", clientIP(ctx))
}
