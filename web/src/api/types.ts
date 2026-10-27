// Entity shapes mirroring docs/API.md exactly.
export type Role = 'admin' | 'supervisor' | 'agent';

export interface User {
  id: number; username: string; display_name: string; email: string; role: Role;
  extension: string; endpoint: string; skills: string[]; enabled: boolean; created_at: string; updated_at: string;
}
export type UserInput = Partial<User> & { password?: string; queues?: { queue_id: number; penalty: number }[] };

export type AgentStatus = 'offline' | 'available' | 'ringing' | 'busy' | 'wrapup' | 'paused';
export interface AgentState {
  user_id: number; username: string; display_name: string; extension: string;
  status: AgentStatus; pause_reason?: string; since: string; call_id?: string; last_call_end: string;
  calls_today: number; talk_secs_today: number; missed_in_row: number; wrapup_until?: string; queue_ids: number[];
}

export type ActionType = '' | 'queue' | 'ai' | 'voicemail' | 'callback' | 'hangup' | 'flow' | 'agent' | 'external' | 'dialplan';
export interface Action { type: ActionType; target?: string; media?: string }

export type QueueStrategy = 'ringall' | 'longest_idle' | 'round_robin' | 'fewest_calls' | 'random';
export interface Queue {
  id: number; name: string; description: string; mode: 'native' | 'asterisk'; extension: string;
  strategy: QueueStrategy; priority: number; ring_timeout_sec: number; max_wait_sec: number; wrapup_sec: number;
  sla_sec: number; moh_class: string; join_media: string; required_skills: string[]; record: boolean;
  allow_callback: boolean; leave_when_empty: boolean; survey: boolean;
  /** 0 = off */ alert_waiting: number; /** 0 = off */ alert_wait_sec: number; overflow: Action; enabled: boolean;
}
export type CampaignStatus = 'draft' | 'running' | 'paused' | 'completed';
export interface Campaign {
  id: number; name: string; description: string; queue_id: number; status: CampaignStatus; max_attempts: number;
  retry_minutes: number; schedule_id: number; script: string;
}
export type LeadStatus = 'pending' | 'dialing' | 'answered' | 'failed' | 'done';
export interface Lead {
  id: number; campaign_id: number; number: string; name: string; status: LeadStatus; attempts: number;
  next_attempt: string; call_id: string; notes: string; updated_at: string;
}
export interface CampaignStats { leads: Partial<Record<LeadStatus | 'total', number>>; calls: Summary }
export interface QueueMember { queue_id: number; user_id: number; penalty: number }

export interface IVRFlow {
  id: number; name: string; dnis: string; greeting_media: string; invalid_media: string;
  timeout_sec: number; max_retries: number; options: Record<string, Action>; timeout_action: Action;
  direct_action: Action; schedule_id: number; after_hours_action: Action; enabled: boolean;
}
export interface WeeklyRange { day: number; start: string; end: string }
export interface Holiday { date: string; name: string }
export interface Schedule { id: number; name: string; timezone: string; weekly: WeeklyRange[]; holidays: Holiday[] }

export interface AIProfile {
  id: number; name: string; model: string; voice: string; language: string; instructions: string;
  greeting: string; temperature: number; tools: string[]; handoff_queue_id: number; max_duration_sec: number;
  knowledge_base: string; enabled: boolean;
}
export interface Disposition { id: number; code: string; label: string; enabled: boolean }
export interface PauseReason { id: number; label: string; enabled: boolean }
export interface Webhook { id: number; url: string; events: string[]; secret: string; enabled: boolean }
export interface Contact { id: number; name: string; phone: string; email: string; company: string; notes: string; tags: string[]; vip: boolean }

export type CallState = 'ivr' | 'ai' | 'queued' | 'ringing' | 'connected' | 'hold' | 'voicemail' | 'dialing' | 'message' | 'survey' | 'ended';
export interface Call {
  id: string; direction: 'inbound' | 'outbound'; caller_number: string; caller_name: string; dnis: string;
  state: CallState; outcome: string; flow_id: number; queue_id: number; agent_id: number; ai_profile_id: number;
  contact_id: number; ivr_path: string; started_at: string; queued_at?: string; answered_at?: string; ended_at?: string;
  wait_sec: number; talk_sec: number; hold_sec: number; sla_met: boolean; disposition: string; notes: string;
  tags: string[]; recording: string; summary: string; sentiment: string; hangup_by: string;
  /** 0 = not rated, else 1-5 */
  csat: number; campaign_id: number;
}
export interface CallView extends Call { ringing_agents: number[]; waiting_sec: number; duration_sec: number; monitors: number }
export interface QueueView {
  id: number; name: string; waiting: number; longest_wait_sec: number; available: number;
  busy: number; paused: number; logged_in: number; members: number;
}
export interface TranscriptEntry { call_id: string; speaker: 'caller' | 'ai' | 'agent' | 'call'; text: string; at: string }
export interface CallEvent { call_id: string; type: string; detail: string; at: string }
export type CallbackStatus = 'pending' | 'in_progress' | 'done' | 'failed' | 'cancelled';
export interface Callback {
  id: number; call_id: string; number: string; name: string; queue_id: number; status: CallbackStatus;
  assigned_to: number; due_at?: string; attempts: number; notes: string; created_at: string; updated_at: string;
}
export interface Voicemail {
  id: number; call_id: string; caller_number: string; caller_name: string; queue_id: number;
  recording: string; duration_sec: number; transcript: string; heard: boolean; created_at: string;
}
export interface AuditEntry { id: number; user_id: number; username: string; action: string; entity: string; entity_id: string; detail: string; at: string }
export interface Summary {
  total: number; inbound: number; outbound: number; queued: number; answered: number; abandoned: number;
  voicemail: number; callbacks: number; ai_resolved: number; sla_met: number; sla_percent: number; abandon_rate: number;
  avg_wait_sec: number; avg_talk_sec: number; max_wait_sec: number; total_talk_sec: number;
  csat_count: number; csat_avg: number;
}
export interface GroupStat extends Summary { id: number }
export interface AgentTime { user_id: number; seconds: Record<string, number> }
export interface Bucket { key: string; total: number; answered: number; abandoned: number }
export interface LabelCount { label: string; count: number }

export interface Snapshot { calls: CallView[]; agents: AgentState[]; queues: QueueView[]; ai_sessions: number; alerts?: QueueAlert[]; at: string }
export interface LoginResponse { token: string; expires_at: string; user: User }
export interface MeResponse { user: User; agent: AgentState | null }
export interface CallDetail { call: Call; live: CallView | null; transcript: TranscriptEntry[]; events: CallEvent[]; contact: Contact | null }
export interface Paged<T> { items: T[]; total: number }
export interface AIStatus { enabled: boolean; model: string; voice: string; voices: string[]; tools: string[] }
export interface ClientSecret { value: string; expires_at: string; model: string }
export interface SystemInfo {
  version: string; ari_connected: boolean; ai_enabled: boolean; db_ok: boolean; uptime_sec: number; ws_clients: number;
  default_queue: string; queue_context: string; outbound_template: string; started_at: string;
}
export interface QueueAlert { queue_id: number; name: string; level: 'alert' | 'clear'; waiting: number; longest_wait_sec: number; available: number }
export interface ScreenPop { user_id: number; call: CallView }
export interface WsMessage { type: string; data: unknown; at: string }

export const OUTCOMES = ['answered', 'abandoned', 'voicemail', 'callback_requested', 'ai_resolved', 'sent_to_dialplan',
  'failed', 'no_answer', 'ivr_hangup', 'transferred_external'] as const;
export const EVENT_TYPES = ['call.updated', 'call.ended', 'agent.updated', 'queue.updated', 'call.transcript',
  'agent.screenpop', 'callback.created', 'voicemail.created', 'config.changed', 'queue.alert'] as const;
