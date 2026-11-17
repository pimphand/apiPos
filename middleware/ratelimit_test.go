package middleware

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoginRateLimitPerIP(t *testing.T) {
	limiter := newRateLimiter(authRateLimits)
	const ip = "203.0.113.10"

	for i := 0; i < 10; i++ {
		assert.True(t, limiter.allow(ip, "/auth.v1.AuthService/Login"), "call %d must be allowed", i)
	}
	assert.False(t, limiter.allow(ip, "/auth.v1.AuthService/Login"), "11th login from one IP must be rejected")
	assert.True(t, limiter.allow("203.0.113.11", "/auth.v1.AuthService/Login"), "other IPs are unaffected")
}

func TestRegisterRateLimitPerIP(t *testing.T) {
	limiter := newRateLimiter(authRateLimits)
	const ip = "198.51.100.7"

	for i := 0; i < 5; i++ {
		assert.True(t, limiter.allow(ip, "/auth.v1.AuthService/Register"), "call %d must be allowed", i)
	}
	assert.False(t, limiter.allow(ip, "/auth.v1.AuthService/Register"), "6th registration from one IP must be rejected")
}

func TestRateLimitLeavesUnlistedMethodsAlone(t *testing.T) {
	limiter := newRateLimiter(authRateLimits)
	const ip = "198.51.100.8"

	for i := 0; i < 100; i++ {
		assert.True(t, limiter.allow(ip, "/sale.v1.SaleService/CreateSale"))
	}
}
