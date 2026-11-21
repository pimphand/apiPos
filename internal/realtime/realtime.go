// Package realtime provides a lightweight WebSocket broadcast hub used to
// push product, inventory and sale changes to connected clients (POS, product
// and inventory pages) in real time.
//
// The hub is store-scoped: every message carries a store_id and is delivered
// only to clients connected under that store. Messages are fire-and-forget —
// clients refetch their data from the API when a relevant event arrives, so
// the hub never stores business state.
package realtime

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"apiPos/internal/apperror"
	"apiPos/internal/auth"
)

// Event types published to WebSocket clients.
// Keep these in sync with the frontend (fe/src/lib/realtime.ts).
const (
	EventProductCreated    = "product.created"
	EventProductUpdated    = "product.updated"
	EventProductDeleted    = "product.deleted"
	EventProductSKUCreated = "productsku.created"
	EventProductSKUUpdated = "productsku.updated"
	EventProductSKUDeleted = "productsku.deleted"
	EventInventoryAdjusted = "inventory.adjusted"
	EventSaleCreated       = "sale.created"
	EventSaleCancelled     = "sale.cancelled"
	EventSaleRefunded      = "sale.refunded"
)

// Event is a message pushed to subscribed WebSocket clients.
type Event struct {
	Event      string `json:"event"`
	StoreID    string `json:"store_id"`
	Resource   string `json:"resource,omitempty"`
	ResourceID string `json:"resource_id,omitempty"`
	Data       any    `json:"data,omitempty"`
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
	sendBufferSize = 64
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	Subprotocols:    []string{"pos-realtime"},
	// The browser connects to the WS endpoint through the same origin as the
	// app (Vite dev proxy / nginx). JWT auth is enforced below for every
	// connection, so checking the origin is not required for security.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Client represents a single connected WebSocket client.
type Client struct {
	hub       *Hub
	conn      *websocket.Conn
	send      chan []byte
	storeID   string
	userID    string
	expiresAt time.Time
}

// Hub manages connected clients and routes events to the right store.
type Hub struct {
	mu      sync.RWMutex
	clients map[*Client]bool
}

// NewHub creates an empty hub.
func NewHub() *Hub {
	return &Hub{clients: make(map[*Client]bool)}
}

// defaultHub is the process-wide hub used by the application services.
var defaultHub = NewHub()

// Default returns the process-wide hub.
func Default() *Hub {
	return defaultHub
}

// Publish broadcasts an event to every client connected under the event's
// store. It never blocks on slow clients: over-buffered clients are dropped.
func Publish(ev Event) {
	defaultHub.publish(ev)
}

// PublishFromContext publishes an event using the store_id carried by the
// authenticated context claims when the event does not already specify one.
// This lets services broadcast without re-deriving the tenant themselves.
func PublishFromContext(ctx context.Context, ev Event) {
	if ev.StoreID == "" {
		if claims, ok := auth.ClaimsFromContext(ctx); ok && claims != nil {
			ev.StoreID = claims.StoreID
		}
	}
	Publish(ev)
}

// Publish broadcasts an event to every client connected under the event's store.
func (h *Hub) Publish(ev Event) {
	h.publish(ev)
}

func (h *Hub) publish(ev Event) {
	data, err := json.Marshal(ev)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		if c.storeID == ev.StoreID && time.Now().Before(c.expiresAt) {
			select {
			case c.send <- data:
			default:
				// Slow client: drop the message rather than blocking the hub.
			}
		}
	}
}

// register adds a client to the hub.
func (h *Hub) register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c] = true
}

// unregister removes a client from the hub and closes its send channel.
func (h *Hub) unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[c]; ok {
		delete(h.clients, c)
		close(c.send)
	}
}

// readPump drains inbound (control) messages so the connection stays alive and
// disconnect is detected. Clients never send meaningful data, so anything
// received is discarded.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister(c)
		_ = c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}

// writePump delivers queued events and sends periodic pings.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	expiry := time.NewTimer(time.Until(c.expiresAt))
	defer expiry.Stop()
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-expiry.C:
			return
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

const (
	ticketTTL  = 30 * time.Second
	maxTickets = 4096
)

type ticket struct {
	claims *auth.Claims
	until  time.Time
}

type Gateway struct {
	hub     *Hub
	auth    *auth.AuthService
	mu      sync.Mutex
	tickets map[string]ticket
}

func NewGateway(hub *Hub, authSvc *auth.AuthService) *Gateway {
	return &Gateway{hub: hub, auth: authSvc, tickets: make(map[string]ticket)}
}

// IssueTicket authenticates the browser over HTTP before issuing a single-use WebSocket credential.
func (g *Gateway) IssueTicket(w http.ResponseWriter, r *http.Request) {
	claims, err := g.auth.ClaimsFromRequest(r)
	if err != nil {
		if errors.Is(err, apperror.ErrInvalidToken) || errors.Is(err, apperror.ErrTokenExpired) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"code": "UNAUTHENTICATED", "message": "Unauthorized"})
		} else {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"code": "INTERNAL", "message": "Could not validate session"})
		}
		return
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"code": "INTERNAL", "message": "Could not issue ticket"})
		return
	}
	key := base64.RawURLEncoding.EncodeToString(secret)
	now := time.Now()
	expires := now.Add(ticketTTL)
	if claims.ExpiresAt.Before(expires) {
		expires = claims.ExpiresAt.Time
	}
	g.mu.Lock()
	for candidate, entry := range g.tickets {
		if !now.Before(entry.until) {
			delete(g.tickets, candidate)
		}
	}
	if len(g.tickets) >= maxTickets {
		g.mu.Unlock()
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"code": "RESOURCE_EXHAUSTED", "message": "Too many outstanding tickets"})
		return
	}
	g.tickets[key] = ticket{claims: claims, until: expires}
	g.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]string{"ticket": key})
}

func writeJSON(w http.ResponseWriter, status int, data map[string]string) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (g *Gateway) consume(key string) *auth.Claims {
	g.mu.Lock()
	defer g.mu.Unlock()
	entry, found := g.tickets[key]
	delete(g.tickets, key)
	if !found || !time.Now().Before(entry.until) {
		return nil
	}
	return entry.claims
}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var claims *auth.Claims
	for _, protocol := range websocket.Subprotocols(r) {
		if key, ok := strings.CutPrefix(protocol, "ticket."); ok {
			claims = g.consume(key)
			break
		}
	}
	if claims == nil {
		claims, _ = g.auth.ClaimsFromRequest(r)
	}
	if claims == nil {
		http.Error(w, `{"code":"UNAUTHENTICATED","message":"invalid or missing token"}`, http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("realtime: websocket upgrade failed: %v", err)
		return
	}

	client := &Client{
		hub: g.hub, conn: conn, send: make(chan []byte, sendBufferSize),
		storeID: claims.StoreID, userID: claims.UserID, expiresAt: claims.ExpiresAt.Time,
	}
	g.hub.register(client)
	go client.writePump()
	go client.readPump()
}
