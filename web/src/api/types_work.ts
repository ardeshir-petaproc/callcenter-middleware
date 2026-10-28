// Tickets, knowledge base and email connectors (docs/API_PLATFORM.md).
import type { Call } from './types';

export type TicketStatus = 'new' | 'assigned' | 'in_progress' | 'waiting_customer' | 'resolved' | 'closed';
export type TicketPriority = 'low' | 'normal' | 'high' | 'urgent';
export const TICKET_STATUSES: TicketStatus[] = ['new', 'assigned', 'in_progress', 'waiting_customer', 'resolved', 'closed'];
export const TICKET_PRIORITIES: TicketPriority[] = ['low', 'normal', 'high', 'urgent'];

export interface Ticket {
  id: number; number: string; subject: string; description: string; category: string; priority: TicketPriority;
  status: TicketStatus; source: 'call' | 'email' | 'manual' | 'ai' | string; department_id: number; assignee_id: number;
  contact_id: number; requester_name: string; requester_phone: string; requester_email: string; call_ids: string[] | null;
  tags: string[] | null; created_by: number; created_at: string; updated_at: string; first_response_at?: string;
  resolved_at?: string; closed_at?: string; response_due_at?: string; resolution_due_at?: string;
}
export interface TicketComment { id: number; ticket_id: number; author_id: number; author_name: string; body: string; internal: boolean; created_at: string }
export interface TicketEvent { id: number; ticket_id: number; type: string; from: string; to: string; actor_id: number; actor_name: string; at: string }
export interface TicketDetail { ticket: Ticket; comments: TicketComment[] | null; history: TicketEvent[] | null; calls: Call[] | null; overdue: boolean }
export interface SLAPolicy { id: number; priority: TicketPriority; first_response_minutes: number; resolution_minutes: number }
export type TicketInput = Partial<Omit<Ticket, 'id' | 'number'>>;

export type KBKind = 'file' | 'text' | 'email';
export type KBStatus = 'active' | 'draft' | 'archived';
export interface KnowledgeSource {
  id: number; title: string; kind: KBKind; category: string; tags: string[] | null; department_ids: number[] | null;
  queue_ids: number[] | null; job_profile_ids: number[] | null; topics: string[] | null; visibility: 'all' | 'restricted';
  /** "admin" | "supervisor" | "agent" | "role:<id>" */ allowed_roles: string[] | null; ai_enabled: boolean; status: KBStatus;
  /** YYYY-MM-DD, empty = no expiry */ valid_until: string; owner_id: number; version: number; updated_at: string;
  file_name: string; mime_type: string; size: number; checksum: string; text_chars: number; connector_id: number; email_id: number;
}
export interface KBHit { source_id: number; title: string; category: string; chunk?: number; snippet: string; text?: string; score: number }
export interface KBCategory { category: string; count: number }

export type EmailSecurity = 'tls' | 'starttls' | 'none';
export interface EmailConnector {
  id: number; name: string; host: string; port: number; security: EmailSecurity; username: string; password?: string;
  folder: string; from_filter: string[] | null; subject_filter: string; since_date: string; mode: 'ticket' | 'kb';
  department_id: number; ticket_category: string; kb_category: string; ai_enabled: boolean; poll_minutes: number;
  enabled: boolean; consent_by: number; consent_at: string; status: 'never' | 'ok' | 'error' | string; last_sync: string;
  last_error: string; imported: number; has_password: boolean;
  /** Optional outgoing mail: public ticket replies are emailed to the customer when set. */
  smtp_host?: string; smtp_port?: number; smtp_security?: EmailSecurity; from_address?: string;
}
export interface EmailMessage {
  id: number; connector_id: number; uid: number; message_id: string; from_addr: string; from_name: string;
  subject: string; date: string; ticket_id: number; kb_source_id: number; created_at: string;
}
