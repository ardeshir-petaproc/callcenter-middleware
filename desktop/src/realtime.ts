/**
 * Reconnecting WebSocket client for `/api/v1/ws?token=` running in the main
 * process. Uses the runtime's global WebSocket (Node >= 22, which Electron's
 * main process provides) and falls back to the `ws` package otherwise.
 */
import type { WsMessage } from './server';

const OPEN = 1;

interface MessageLike {
  data: unknown;
}
interface CloseLike {
  code?: number;
  reason?: string;
}

/** The subset of the WHATWG WebSocket API shared by Node's global and `ws`. */
interface SocketLike {
  readonly readyState: number;
  binaryType: string;
  close(code?: number, reason?: string): void;
  addEventListener(type: 'open', listener: () => void): void;
  addEventListener(type: 'message', listener: (ev: MessageLike) => void): void;
  addEventListener(type: 'close', listener: (ev: CloseLike) => void): void;
  addEventListener(type: 'error', listener: (ev: unknown) => void): void;
}

type SocketCtor = new (url: string) => SocketLike;

let cachedCtor: { ctor: SocketCtor; impl: string } | null = null;

function socketCtor(): { ctor: SocketCtor; impl: string } {
  if (cachedCtor) return cachedCtor;
  const g = (globalThis as unknown as { WebSocket?: SocketCtor }).WebSocket;
  if (typeof g === 'function') {
    cachedCtor = { ctor: g, impl: 'global WebSocket' };
  } else {
    // eslint-disable-next-line @typescript-eslint/no-require-imports
    const ws = require('ws') as SocketCtor;
    cachedCtor = { ctor: ws, impl: 'ws package' };
  }
  return cachedCtor;
}

function decode(data: unknown): string | null {
  if (typeof data === 'string') return data;
  if (Buffer.isBuffer(data)) return data.toString('utf8');
  if (data instanceof ArrayBuffer) return Buffer.from(data).toString('utf8');
  if (Array.isArray(data)) return Buffer.concat(data as Buffer[]).toString('utf8');
  return null;
}

export interface RealtimeOptions {
  onMessage(msg: WsMessage): void;
  onConnectionChange?(connected: boolean): void;
  log?(...args: unknown[]): void;
  /** Reconnect when nothing was received for this long (server sends a snapshot every 10 s). */
  idleTimeoutMs?: number;
  maxBackoffMs?: number;
}

export class RealtimeClient {
  private url: string | null = null;
  private socket: SocketLike | null = null;
  private attempt = 0;
  private reconnectTimer: NodeJS.Timeout | null = null;
  private idleTimer: NodeJS.Timeout | null = null;
  private connected = false;

  constructor(private readonly opts: RealtimeOptions) {}

  get isConnected(): boolean {
    return this.connected;
  }

  /** Connects to `url` (no-op when already connected/connecting to it). */
  start(url: string): void {
    if (this.url === url && (this.socket || this.reconnectTimer)) return;
    this.stop();
    this.url = url;
    this.attempt = 0;
    this.connect();
  }

  stop(): void {
    this.url = null;
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
    this.reconnectTimer = null;
    this.clearIdle();
    const s = this.socket;
    this.socket = null;
    if (s) {
      try {
        s.close(1000, 'client stop');
      } catch {
        /* ignore */
      }
    }
    this.setConnected(false);
  }

  private log(...args: unknown[]): void {
    this.opts.log?.(...args);
  }

  private setConnected(v: boolean): void {
    if (this.connected === v) return;
    this.connected = v;
    this.opts.onConnectionChange?.(v);
  }

  private connect(): void {
    const url = this.url;
    if (!url) return;
    const { ctor, impl } = socketCtor();
    let socket: SocketLike;
    try {
      socket = new ctor(url);
    } catch (err) {
      this.log('ws: cannot create socket:', (err as Error).message);
      this.scheduleReconnect();
      return;
    }
    this.socket = socket;
    try {
      socket.binaryType = 'arraybuffer';
    } catch {
      /* ignore */
    }
    this.log(`ws: connecting (${impl}) to`, redact(url));
    this.touch(); // also acts as the handshake timeout

    socket.addEventListener('open', () => {
      if (this.socket !== socket) return;
      this.attempt = 0;
      this.setConnected(true);
      this.touch();
      this.log('ws: connected');
    });
    socket.addEventListener('message', (ev) => {
      if (this.socket !== socket) return;
      this.touch();
      const text = decode(ev.data);
      if (!text) return;
      let msg: unknown;
      try {
        msg = JSON.parse(text);
      } catch {
        return;
      }
      if (msg && typeof msg === 'object' && typeof (msg as WsMessage).type === 'string') {
        try {
          this.opts.onMessage(msg as WsMessage);
        } catch (err) {
          this.log('ws: message handler failed:', (err as Error).message);
        }
      }
    });
    socket.addEventListener('error', () => {
      // A 'close' event always follows; reconnect is handled there.
    });
    socket.addEventListener('close', (ev) => {
      if (this.socket !== socket) return;
      this.socket = null;
      this.clearIdle();
      this.setConnected(false);
      this.log(`ws: closed (code ${ev.code ?? '?'})`);
      this.scheduleReconnect();
    });
  }

  private scheduleReconnect(): void {
    if (!this.url || this.reconnectTimer) return;
    const max = this.opts.maxBackoffMs ?? 30000;
    const base = Math.min(max, 1000 * 2 ** Math.min(this.attempt, 10));
    const delay = Math.round(base * (0.75 + Math.random() * 0.5));
    this.attempt++;
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      this.connect();
    }, delay);
  }

  private touch(): void {
    this.clearIdle();
    const ms = this.opts.idleTimeoutMs ?? 45000;
    this.idleTimer = setTimeout(() => {
      const s = this.socket;
      if (!s) return;
      this.log(`ws: ${s.readyState === OPEN ? 'no traffic' : 'no handshake'} for ${ms / 1000} s, reconnecting`);
      // Detach first: a dead peer may never complete the closing handshake.
      this.socket = null;
      try {
        s.close(4000, 'idle');
      } catch {
        /* ignore */
      }
      this.setConnected(false);
      this.scheduleReconnect();
    }, ms);
  }

  private clearIdle(): void {
    if (this.idleTimer) clearTimeout(this.idleTimer);
    this.idleTimer = null;
  }
}

/** Hides the token when logging a WebSocket URL. */
export function redact(url: string): string {
  return url.replace(/([?&]token=)[^&]+/, '$1***');
}
