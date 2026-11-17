package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateRejectsEmptyOrPlaceholderSecret(t *testing.T) {
	for _, secret := range []string{"", "   ", "change-me", "Change-Me", "changeme", "jwt-secret"} {
		cfg := &Config{JWT: JWTConfig{Secret: secret}}
		require.Error(t, cfg.Validate(), "secret %q must be rejected", secret)
	}
}

func TestValidateRejectsShortSecret(t *testing.T) {
	cfg := &Config{JWT: JWTConfig{Secret: strings.Repeat("a", minJWTSecretBytes-1)}}

	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least 32 bytes")
}

func TestValidateAcceptsLongSecret(t *testing.T) {
	cfg := &Config{JWT: JWTConfig{Secret: strings.Repeat("a", minJWTSecretBytes)}}
	assert.NoError(t, cfg.Validate())
}

func TestValidateTrimsSurroundingWhitespace(t *testing.T) {
	cfg := &Config{JWT: JWTConfig{Secret: "  " + strings.Repeat("a", 40) + "  "}}
	assert.NoError(t, cfg.Validate())
}
