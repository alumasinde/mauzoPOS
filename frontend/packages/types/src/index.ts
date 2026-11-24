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
  /** Tenant role display name, e.g. "Administrator". */
  role_name?: string;
  mfa_enabled: boolean;
  tenant?: TenantRef;
  /** Effective permissions (tenant users: module-gated). UI hint only — the API enforces. */
  permissions: string[];
  /** Tenant users: enabled module codes. */
  modules?: string[];
  all_branches: boolean;
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

export interface BusinessType {
  code: string;
  name: string;
}

export interface ProfileDefinition {
  modules: string[];
  categories: string[];
  settings: Record<string, unknown>;
  receipt_template: string;
  pos_layout: Record<string, unknown>;
}

export interface BusinessProfile {
  id: string;
  business_type: string;
  name: string;
  is_default: boolean;
  definition: ProfileDefinition;
  updated_at: string;
}

export interface Plan {
  id: string;
  name: string;
  description?: string;
  price_kes: MoneyString;
  billing_interval: 'monthly' | 'yearly';
  modules: string[];
  max_branches?: number;
  max_users?: number;
  is_active: boolean;
  sort: number;
}

export interface SubscriptionInfo {
  plan_id: string;
  plan_name: string;
  status: 'trialing' | 'active' | 'past_due' | 'cancelled';
  starts_at: string;
  trial_ends_at?: string;
}

export interface Branch {
  id: string;
  code: string;
  name: string;
  phone?: string;
  location?: string;
  is_main: boolean;
  status: 'active' | 'closed';
}

export interface PosTerminal {
  id: string;
  branch_id: number;
  branch?: string;
  name: string;
  status: 'pending' | 'active' | 'disabled';
  device_name?: string | null;
  activated_at?: string | null;
  last_seen_at?: string | null;
  online?: boolean;
  /** Set while an activation code is waiting to be used. */
  code_expires_at?: string | null;
}

/** A freshly issued activation code — shown once. */
export interface IssuedTerminalCode {
  code: string;
  expires_at: string;
  /** The terminal already has a device: using the code moves it to a new PC. */
  repair: boolean;
}

export interface SettingEntry {
  key: string;
  value: unknown;
  updated_at: string;
}

export interface OnboardingState {
  step: number;
  done: boolean;
  business_type?: string;
  business_types: BusinessType[];
  profiles: BusinessProfile[];
  plans: Plan[];
  modules: string[];
  branches: Branch[];
  terminals: PosTerminal[];
  subscription?: SubscriptionInfo;
}

export * from './phase3';
export * from './catalog';
export * from './inventory';
export * from './pricing';
export * from './sales';
export * from './receipts';
