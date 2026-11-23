import type { MoneyString } from './index';

// ─── Phase 3: modules, roles, employees ─────────────────────────────────────

/** One entry of the dynamic back-office sidebar (GET /t/navigation). */
export interface NavItem {
  key: string;
  label: string;
  path: string;
  icon: string;
}

export type ModuleGroup = 'core' | 'operations' | 'finance' | 'industry' | 'growth';

/** A JSON-schema subset describing a module's config (GET /modules). */
export interface ConfigSchema {
  type: string;
  properties?: Record<string, ConfigSchema>;
  items?: ConfigSchema;
  enum?: (string | number)[];
  minimum?: number;
  maximum?: number;
  minLength?: number;
  maxLength?: number;
  maxItems?: number;
}

export interface ModuleInfo {
  code: string;
  name: string;
  description: string;
  group: ModuleGroup;
  core: boolean;
  depends: string[];
  recommended_for: string[];
  permissions: string[];
  has_config: boolean;
  config_schema?: ConfigSchema;
  /** Tenant view only. */
  enabled?: boolean;
  effective?: boolean;
  source?: 'tenant' | 'platform' | '';
  reason?: 'platform_inactive' | 'not_in_plan' | 'dependency_off' | '';
  in_plan?: boolean;
  platform_active?: boolean;
  config?: Record<string, unknown>;
}

export interface ModuleToggleResult {
  changed: string[];
  modules: ModuleInfo[];
}

export interface Role {
  id: string;
  code: string;
  name: string;
  description: string;
  is_system: boolean;
  is_locked: boolean;
  customized: boolean;
  user_count: number;
  permissions: string[];
  max_discount_percent: string | null;
  max_refund_amount: MoneyString | null;
  max_cash_out_amount: MoneyString | null;
}

export interface PermissionItem {
  code: string;
  description: string;
  sensitive: boolean;
}

export interface PermissionGroup {
  key: string;
  name: string;
  module?: string;
  enabled: boolean;
  permissions: PermissionItem[];
}

export interface Ref {
  id: string;
  name: string;
  code?: string;
}

export interface Employee {
  id: string;
  name: string;
  email: string | null;
  username: string | null;
  role: Ref;
  all_branches: boolean;
  branches: Ref[];
  status: 'active' | 'disabled';
  has_password: boolean;
  has_pin: boolean;
  mfa_enabled: boolean;
  last_login_at: string | null;
  created_at: string;
}
