// Routing, quality assurance and retention (docs/API_PLATFORM.md).
import type { Action } from './types';

export interface Topic { id: number; name: string; description: string; keywords: string[] | null; department_id: number }

export type RuleStatus = 'draft' | 'approved' | 'rejected' | 'retired';
export type TargetType = 'user' | 'job_profile' | 'queue' | 'department';
export interface RoutingRule {
  id: number; name: string; topics: string[] | null; keywords: string[] | null; languages: string[] | null; vip_only: boolean;
  target_type: TargetType; target_id: number; fallback: Action; priority: number; rationale: string; status: RuleStatus;
  source: 'manual' | 'ai' | string; job_profile_id: number; version: number; created_by: number; approved_by: number;
  approved_at: string; updated_at: string;
}
export interface RouteRequest { text: string; topic?: string; language?: string; caller_number?: string; vip?: boolean; call_id?: string }
export interface RouteCandidate { rule_id: number; rule_name: string; score: number; matched: string[] | null; result: 'selected' | 'unavailable' | 'skipped' | string }
export interface RoutingDecision {
  id?: number; call_id: string; at: string; request: RouteRequest; topic: string; rule_id: number; action: Action;
  target_name: string; reason: string; steps: string[] | null; candidates: RouteCandidate[] | null;
}
export interface GenerateResult { method: 'ai' | 'template'; rules: RoutingRule[] }

export interface QACriterion { key: string; label: string; weight: number; max_score: number; critical: boolean }
export interface QAScorecard { id: number; name: string; description: string; pass_score: number; criteria: QACriterion[]; enabled: boolean }
export interface QAEvaluation {
  id: number; call_id: string; agent_id: number; scorecard_id: number; evaluator_id: number;
  scores: Record<string, number> | null; notes: Record<string, string> | null; comment: string; score: number;
  auto_failed: boolean; passed: boolean; status: 'draft' | 'final'; acknowledged: boolean; agent_comment: string;
  created_at: string; updated_at: string;
}
export interface QAEvaluationDetail { evaluation: QAEvaluation; scorecard: QAScorecard | null }
export interface QASummaryRow { agent_id: number; evaluations: number; avg_score: number; passed: number; auto_failed: number }
export interface QASummary { agents: QASummaryRow[]; scale?: string }

export interface RetentionPolicy { recording_days: number; transcript_days: number; email_days: number }
export interface RetentionResult { recordings: number; transcripts: number; emails: number; failed: number }
