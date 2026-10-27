/**
 * The main process' own view of the logged-in agent.
 *
 * The web panel owns authentication: it stores its bearer token in
 * localStorage (`cc_token`). main.ts polls that value and hands it to
 * AgentSession.setToken(); from there we resolve the user via /auth/me, keep a
 * WebSocket open for realtime events (incoming-call notifications, own status)
 * and perform tray status changes through POST /me/status.
 */
import { RealtimeClient } from './realtime';
import {
  ApiError,
  apiRequest,
  wsUrl,
  type AgentState,
  type AgentStatus,
  type CallView,
  type MeResponse,
  type PauseReason,
  type QueueView,
  type SettableStatus,
  type User,
  type WsMessage,
} from './server';

export interface AgentSessionHooks {
  /** Something shown in the tray changed (user, status, connection). */
  onChange(): void;
  /** The logged-in user's phone is ringing. */
  onScreenpop(call: CallView, queueName: string | null): void;
  log(...args: unknown[]): void;
}

const ME_RETRY_MS = 15000;
const VERIFY_DEBOUNCE_MS = 20000;
const SCREENPOP_DEDUPE_MS = 60000;

export class AgentSession {
  private serverUrl: string | null = null;
  private token: string | null = null;
  /** Token that the server rejected with 401; not retried until it changes. */
  private rejectedToken: string | null = null;
  private generation = 0;
  private loading = false;
  private nextRetryAt = 0;
  private verifyTimer: NodeJS.Timeout | null = null;
  private readonly queueNames = new Map<number, string>();
  private readonly recentPops = new Map<string, number>();
  private readonly rt: RealtimeClient;

  user: User | null = null;
  status: AgentStatus | null = null;
  pauseReason: string | null = null;
  pauseReasons: PauseReason[] = [];

  constructor(private readonly hooks: AgentSessionHooks) {
    this.rt = new RealtimeClient({
      onMessage: (m) => this.onMessage(m),
      onConnectionChange: (up) => {
        if (!up) this.scheduleVerify();
        this.hooks.onChange();
      },
      log: (...a) => this.hooks.log(...a),
    });
  }

  get connected(): boolean {
    return this.rt.isConnected;
  }

  get loggedIn(): boolean {
    return this.user !== null;
  }

  setServer(serverUrl: string | null): void {
    if (serverUrl === this.serverUrl) return;
    this.serverUrl = serverUrl;
    this.setToken(null);
    this.rejectedToken = null;
  }

  /** Called periodically with the panel's current token (null when logged out). */
  setToken(token: string | null): void {
    if (token === this.token) {
      if (token && !this.user && !this.loading && token !== this.rejectedToken && Date.now() >= this.nextRetryAt) {
        void this.loadMe();
      }
      return;
    }
    this.token = token;
    this.generation++;
    this.loading = false;
    this.user = null;
    this.status = null;
    this.pauseReason = null;
    this.pauseReasons = [];
    this.nextRetryAt = 0;
    this.rt.stop();
    if (this.verifyTimer) clearTimeout(this.verifyTimer);
    this.verifyTimer = null;
    this.hooks.log(token ? 'session: token changed' : 'session: logged out');
    this.hooks.onChange();
    if (token && token !== this.rejectedToken) void this.loadMe();
  }

  stop(): void {
    this.setToken(null);
  }

  private async loadMe(): Promise<void> {
    const server = this.serverUrl;
    const token = this.token;
    if (!server || !token) return;
    const gen = this.generation;
    this.loading = true;
    try {
      const me = await apiRequest<MeResponse>(server, token, 'GET', '/auth/me');
      if (gen !== this.generation) return;
      if (!me || !me.user || typeof me.user.id !== 'number') throw new Error('unexpected /auth/me response');
      this.user = me.user;
      this.applyAgentState(me.agent);
      this.hooks.log(`session: signed in as ${me.user.username} (#${me.user.id}, ${me.user.role})`);
      this.rt.start(wsUrl(server, token));
      this.hooks.onChange();
      void this.loadPauseReasons();
    } catch (err) {
      if (gen !== this.generation) return;
      if (err instanceof ApiError && err.status === 401) {
        this.rejectedToken = token;
        this.hooks.log('session: token rejected by server (401)');
      } else {
        this.nextRetryAt = Date.now() + ME_RETRY_MS;
        this.hooks.log('session: /auth/me failed:', (err as Error).message);
      }
    } finally {
      if (gen === this.generation) this.loading = false;
    }
  }

  private async loadPauseReasons(): Promise<void> {
    const server = this.serverUrl;
    const token = this.token;
    if (!server || !token) return;
    const gen = this.generation;
    try {
      const list = await apiRequest<PauseReason[]>(server, token, 'GET', '/pause-reasons');
      if (gen !== this.generation || !Array.isArray(list)) return;
      this.pauseReasons = list.filter((r) => r && r.enabled !== false && typeof r.label === 'string');
      this.hooks.onChange();
    } catch {
      /* optional feature */
    }
  }

  /** After the socket drops, re-check the token so an expired one stops the reconnect loop. */
  private scheduleVerify(): void {
    if (this.verifyTimer || !this.user) return;
    const gen = this.generation;
    this.verifyTimer = setTimeout(async () => {
      this.verifyTimer = null;
      const server = this.serverUrl;
      const token = this.token;
      if (gen !== this.generation || !server || !token || this.rt.isConnected) return;
      try {
        await apiRequest<MeResponse>(server, token, 'GET', '/auth/me');
      } catch (err) {
        if (gen === this.generation && err instanceof ApiError && err.status === 401) {
          this.hooks.log('session: token expired, stopping realtime connection');
          this.rejectedToken = token;
          this.rt.stop();
          this.user = null;
          this.status = null;
          this.hooks.onChange();
        }
      }
    }, VERIFY_DEBOUNCE_MS);
  }

  /** Changes the logged-in user's own status via POST /me/status. */
  async setStatus(status: SettableStatus, reason?: string): Promise<void> {
    const server = this.serverUrl;
    const token = this.token;
    if (!server || !token || !this.user) throw new Error('Not signed in to the panel.');
    const body: { status: SettableStatus; reason?: string } = { status };
    if (reason) body.reason = reason;
    const state = await apiRequest<AgentState>(server, token, 'POST', '/me/status', body);
    this.applyAgentState(state);
    this.hooks.onChange();
  }

  private applyAgentState(state: AgentState | null | undefined): void {
    if (!state || typeof state.status !== 'string') return;
    this.status = state.status;
    this.pauseReason = state.pause_reason || null;
  }

  private onMessage(msg: WsMessage): void {
    const me = this.user;
    const data = msg.data as Record<string, unknown> | null;
    if (!me || !data || typeof data !== 'object') return;
    switch (msg.type) {
      case 'snapshot': {
        if (Array.isArray(data.queues)) {
          for (const q of data.queues as QueueView[]) {
            if (q && typeof q.id === 'number' && typeof q.name === 'string') this.queueNames.set(q.id, q.name);
          }
        }
        if (Array.isArray(data.agents)) {
          const own = (data.agents as AgentState[]).find((a) => a && a.user_id === me.id);
          if (own) this.updateOwnState(own);
        }
        break;
      }
      case 'agent.updated':
        if (Number(data.user_id) === me.id) this.updateOwnState(data as unknown as AgentState);
        break;
      case 'agent.screenpop': {
        if (Number(data.user_id) !== me.id) break;
        const call = data.call as CallView | undefined;
        if (!call || typeof call !== 'object') break;
        if (!this.firstPop(String(call.id ?? ''))) break;
        const queueName = call.queue_id ? (this.queueNames.get(call.queue_id) ?? null) : null;
        this.hooks.log(`session: screenpop for call ${call.id ?? '?'}`);
        this.hooks.onScreenpop(call, queueName);
        break;
      }
      case 'config.changed':
        if (typeof data.entity === 'string' && data.entity.toLowerCase().includes('pause')) void this.loadPauseReasons();
        break;
      default:
        break;
    }
  }

  private updateOwnState(state: AgentState): void {
    const before = `${this.status}|${this.pauseReason}`;
    this.applyAgentState(state);
    if (`${this.status}|${this.pauseReason}` !== before) this.hooks.onChange();
  }

  /** The server may repeat a screenpop (e.g. re-ring); notify once per call. */
  private firstPop(callId: string): boolean {
    const now = Date.now();
    for (const [id, t] of this.recentPops) if (now - t > SCREENPOP_DEDUPE_MS) this.recentPops.delete(id);
    if (!callId) return true;
    if (this.recentPops.has(callId)) return false;
    this.recentPops.set(callId, now);
    return true;
  }
}
