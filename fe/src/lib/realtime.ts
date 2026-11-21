/**
 * Realtime client: a single WebSocket to the backend `/ws` endpoint, scoped
 * by store (from the JWT). The backend publishes events for product,
 * inventory, SKU and sale changes; clients simply re-fetch on the relevant
 * event since events are fire-and-forget and do not carry full payloads.
 *
 * Connection strategy:
 * - One shared socket per page (module singleton).
 * - Authenticated with a single-use ticket issued over HTTP.
 * - Automatic reconnect with exponential backoff.
 * - `on(event, handler)` subscribes and returns an unsubscribe function.
 */
import { authStorage } from '@/lib/auth';
import http from '@/lib/http';
import { WS_URL } from '@/lib/apiBase';

export interface RealtimeEvent {
  event: string;
  store_id?: string;
  resource: string;
  resource_id: string;
  data?: Record<string, unknown>;
}

export const RealtimeEventType = {
  productCreated: 'product.created',
  productUpdated: 'product.updated',
  productDeleted: 'product.deleted',
  productSKUCreated: 'productsku.created',
  productSKUUpdated: 'productsku.updated',
  productSKUDeleted: 'productsku.deleted',
  inventoryAdjusted: 'inventory.adjusted',
  saleCreated: 'sale.created',
  saleCancelled: 'sale.cancelled',
  saleRefunded: 'sale.refunded',
} as const;

type Handler = (ev: RealtimeEvent) => void;

const RECONNECT_BASE_MS = 1000;
const RECONNECT_MAX_MS = 15000;

class RealtimeClient {
  private ws: WebSocket | null = null;
  private handlers = new Map<string, Set<Handler>>();
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private reconnectAttempts = 0;
  private noTokenTimer: ReturnType<typeof setTimeout> | null = null;
  private noTokenRetries = 0;
  private shouldReconnect = false;
  private manualClose = false;
  private ticketRequest: AbortController | null = null;

  /** Start (or restart) the connection if not open. Idempotent. */
  connect(): void {
    if (typeof window === 'undefined' || this.ticketRequest) return;
    if (this.ws && (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING)) {
      return;
    }

    const token = authStorage.getAccessToken();
    if (!token) {
      // No access token cached yet (e.g. session-restore is still in flight
      // right after a page reload, or login just happened). Never open an
      // unauthenticated socket the server would reject; instead poll briefly
      // for the token to appear. If no refresh token is persisted the user is
      // signed out, so stop retrying until connect() is called again (the auth
      // store calls connect() after login/restore).
      if (!authStorage.hasPersistedRefresh()) return;
      if (this.noTokenTimer) return;
      this.noTokenRetries += 1;
      if (this.noTokenRetries > 20) return; // ~20s cap; auth store calls connect() on success
      this.noTokenTimer = setTimeout(() => {
        this.noTokenTimer = null;
        this.connect();
      }, 1000);
      return;
    }
    this.noTokenRetries = 0;
    this.shouldReconnect = true;
    this.manualClose = false;

    void this.openSocket();
  }

  private async openSocket(): Promise<void> {
    const request = new AbortController();
    this.ticketRequest = request;
    try {
      const { data } = await http.post<{ ticket: string }>('/v1/realtime/ticket', undefined, {
        signal: request.signal,
      });
      if (request.signal.aborted) return;
      this.attachSocket(new WebSocket(WS_URL, ['pos-realtime', `ticket.${data.ticket}`]));
    } catch {
      if (!request.signal.aborted && this.shouldReconnect) this.scheduleReconnect();
    } finally {
      if (this.ticketRequest === request) this.ticketRequest = null;
    }
  }

  private attachSocket(socket: WebSocket): void {
    this.ws = socket;
    socket.onopen = () => {
      this.reconnectAttempts = 0;
    };

    socket.onmessage = (msg) => {
      let ev: RealtimeEvent;
      try {
        ev = JSON.parse(msg.data as string) as RealtimeEvent;
      } catch {
        return;
      }
      const set = this.handlers.get(ev.event);
      if (set) {
        for (const h of set) {
          try {
            h(ev);
          } catch {
            // One bad handler must not break the others.
          }
        }
      }
    };

    socket.onclose = () => {
      if (this.ws !== socket) return;
      this.ws = null;
      if (this.shouldReconnect && !this.manualClose) {
        this.scheduleReconnect();
      }
    };

    socket.onerror = () => {
      // onclose handles reconnect; nothing to do here.
    };
  }

  /** Gracefully close the socket (e.g. on logout). */
  disconnect(): void {
    this.shouldReconnect = false;
    this.manualClose = true;
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    if (this.noTokenTimer) {
      clearTimeout(this.noTokenTimer);
      this.noTokenTimer = null;
    }
    this.noTokenRetries = 0;
    this.ticketRequest?.abort();
    this.ticketRequest = null;
    this.ws?.close();
    this.ws = null;
  }

  /** Subscribe to an event. Returns an unsubscribe function. */
  on(event: string, handler: Handler): () => void {
    let set = this.handlers.get(event);
    if (!set) {
      set = new Set();
      this.handlers.set(event, set);
    }
    set.add(handler);
    return () => {
      set.delete(handler);
    };
  }

  /** Whether the underlying socket is currently open. */
  isRealtimeConnected(): boolean {
    return !!this.ws && this.ws.readyState === WebSocket.OPEN;
  }

  private scheduleReconnect(): void {
    if (this.reconnectTimer) return;
    const delay = Math.min(
      RECONNECT_BASE_MS * 2 ** this.reconnectAttempts,
      RECONNECT_MAX_MS
    );
    this.reconnectAttempts += 1;
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      this.connect();
    }, delay);
  }
}

export const realtime = new RealtimeClient();
