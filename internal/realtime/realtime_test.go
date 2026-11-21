package realtime

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"apiPos/internal/auth"
	"apiPos/internal/config"
)

func TestTicketCanOnlyBeConsumedOnceBeforeExpiry(t *testing.T) {
	gateway := NewGateway(NewHub(), nil)
	claims := &auth.Claims{UserID: "user", StoreID: "store"}
	gateway.tickets["valid"] = ticket{claims: claims, until: time.Now().Add(time.Minute)}
	gateway.tickets["expired"] = ticket{claims: claims, until: time.Now().Add(-time.Minute)}

	require.Same(t, claims, gateway.consume("valid"))
	require.Nil(t, gateway.consume("valid"))
	require.Nil(t, gateway.consume("expired"))
	require.Empty(t, gateway.tickets)
}

func TestUnauthorizedWebSocketResponseKeepsLegacyFormat(t *testing.T) {
	gateway := NewGateway(NewHub(), auth.NewAuthService(&config.Config{}, nil))
	resp := httptest.NewRecorder()
	gateway.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/ws", nil))

	require.Equal(t, http.StatusUnauthorized, resp.Code)
	require.Equal(t, "text/plain; charset=utf-8", resp.Header().Get("Content-Type"))
	require.Equal(t, "{\"code\":\"UNAUTHENTICATED\",\"message\":\"invalid or missing token\"}\n", resp.Body.String())
}
