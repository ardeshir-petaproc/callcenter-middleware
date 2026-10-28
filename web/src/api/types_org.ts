// Organizations, people and access (docs/API_PLATFORM.md).
import type { Action, Role, Tenant, User } from './types';

export interface Department {
  id: number; name: string; description: string; manager_id: number; schedule_id: number;
  /** Max concurrent calls (0 = unlimited). */ capacity: number; priority: number;
  /** Used when closed, over capacity or unstaffed. */ fallback: Action; languages: string[]; enabled: boolean;
}

export interface JobProfile {
  id: number; title: string; department_id: number; description: string; responsibilities: string[];
  request_types: string[]; authority_limits: string; escalation_conditions: string; escalate_to_user_id: number;
  is_template: boolean; version: number; updated_at: string;
}
export interface JobTemplate extends Omit<JobProfile, 'id'> { id?: number; title_fa: string }

export interface PhoneNumber { id: number; number: string; label: string; flow_id: number; enabled: boolean }

export interface CustomRole { id: number; name: string; description: string; permissions: string[] }

export type InviteStatus = 'pending' | 'accepted' | 'revoked';
export interface Invitation {
  id: number; name: string; email: string; phone: string; joow_account_id: string; role: Role; custom_role_id: number;
  department_ids: number[]; status: InviteStatus; expires_at: string; invited_by: number; user_id: number;
}
export interface InviteCreated { invitation: Invitation; link: string; token: string }
export interface InvitePeek {
  organization: string; name: string; email: string; role: Role; expires_at?: string;
  joow_account_id?: string; joow_account_required?: boolean;
}

export interface Revision { id: number; entity: string; entity_id: number; action: string; data: string; changed_by: number; changed_by_name: string; at: string }

export interface JoowAccount { id: string; name: string; email: string; phone: string; avatar_url?: string }
export interface TenantChoice { id: number; slug: string; name: string; user_id: number; role: Role }
export interface ChooseTenant { choose_tenant: true; tenants: TenantChoice[]; account: JoowAccount }

export interface PermissionsResponse { all: string[]; mine: string[] }
export interface TenantCreated { tenant: Tenant; owner: User }

export type RevisionEntity = 'department' | 'job_profile' | 'phone_number' | 'custom_role' | 'user' | 'topic' | 'routing_rule' | 'qa_scorecard';
