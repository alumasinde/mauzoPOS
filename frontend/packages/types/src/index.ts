// Shared API types. Mirrors backend/internal/core/httpx envelopes.
// Module DTO types are added per phase (later generated from OpenAPI).

/** Monetary amounts travel as decimal strings ("750.34"), never JS numbers. */
export type MoneyString = string;

export interface ApiErrorBody {
  code: string;
  message: string;
  details?: Record<string, unknown>;
}

export interface PageMeta {
  page: number;
  per_page: number;
  total: number;
}

export interface Envelope<T> {
  data: T;
  meta?: PageMeta;
}

export interface Paginated<T> {
  data: T[];
  meta: PageMeta;
}

export interface PageQuery {
  page?: number;
  per_page?: number;
  sort?: string;
  q?: string;
}

export interface HealthStatus {
  status: 'ok' | 'degraded';
  version: string;
  checks?: Record<string, 'ok' | 'unavailable'>;
}

// ─── Phase 1: auth, tenants, users, audit ───────────────────────────────────

export interface TenantRef {
  id: string;
  name: string;
  slug: string;
}

/** The signed-in account (GET /auth/me and login responses). */
export interface Profile {
  id: string;
  name: string;
  identity: string;
  role: string;
  mfa_enabled: boolean;
  tenant?: TenantRef;
}

export interface LoginResponse {
  mfa_required?: boolean;
  ticket?: string;
  token?: string;
  user?: Profile;
}

export interface Tenant {
  id: string;
  name: string;
  slug: string;
  email?: string;
  phone?: string;
  status: 'active' | 'suspended';
  suspended_at?: string;
  suspend_reason?: string;
  created_at: string;
}

export interface PlatformUser {
  id: string;
  name: string;
  email: string;
  role: 'super_admin' | 'support' | 'billing' | 'viewer';
  status: 'active' | 'disabled';
  mfa_enabled: boolean;
  last_login_at?: string;
  created_at: string;
}

export interface SessionInfo {
  id: string;
  ip_address?: string;
  user_agent?: string;
  current: boolean;
  created_at: string;
  last_used_at?: string;
}

export interface LoginHistoryEntry {
  identity: string;
  success: boolean;
  failure_reason?: string;
  ip_address?: string;
  user_agent?: string;
  created_at: string;
}

export interface AuditEntry {
  id: number;
  actor_type: string;
  action: string;
  entity_type?: string;
  entity_id?: string;
  data?: Record<string, unknown>;
  ip_address?: string;
  user_agent?: string;
  created_at: string;
}

export interface MFASetup {
  secret: string;
  otpauth_url: string;
}

export interface MFAEnabled {
  recovery_codes: string[];
}
