import { goto } from '$app/navigation';
import { resolve } from '$app/paths';
import type { LoginResponse, Profile } from '@mauzo/types';
import { api } from './api';

const TOKEN_KEY = 'mauzo_web_token';
const SLUG_KEY = 'mauzo_tenant_slug';
export const AUTH_BASE = '/api/v1/t/auth';

/** Public routes that never need a session. */
const PUBLIC_PATHS = new Set(['/login', '/forgot-password', '/reset-password']);

class Auth {
  token = $state<string | null>(null);
  user = $state<Profile | null>(null);
  /** Last-used company slug, prefilled on the login form. */
  tenantSlug = $state('');
  /** True once the first me() check has resolved. */
  ready = $state(false);

  async init(): Promise<void> {
    this.token = localStorage.getItem(TOKEN_KEY);
    this.tenantSlug = localStorage.getItem(SLUG_KEY) ?? '';
    if (this.token) {
      try {
        this.user = await api.get<Profile>(`${AUTH_BASE}/me`);
        if (this.user.tenant) {
          this.tenantSlug = this.user.tenant.slug;
          localStorage.setItem(SLUG_KEY, this.tenantSlug);
        }
      } catch {
        this.clear();
      }
    }
    this.ready = true;
  }

  async login(slug: string, identity: string, password: string): Promise<LoginResponse> {
    const res = await api.post<LoginResponse>(`${AUTH_BASE}/login`, {
      tenant: slug.trim(),
      identity: identity.trim(),
      password,
    });
    if (res?.token) this.setSession(res, slug);
    return res ?? {};
  }

  async verifyMFA(ticket: string, code: string): Promise<void> {
    const res = await api.post<LoginResponse>(`${AUTH_BASE}/mfa/verify`, { ticket, code });
    if (res?.token) this.setSession(res, this.tenantSlug);
  }

  /** Re-fetches /me into auth.user (e.g. after MFA enable/disable). */
  async refresh(): Promise<void> {
    this.user = await api.get<Profile>(`${AUTH_BASE}/me`);
    if (this.user.tenant) this.tenantSlug = this.user.tenant.slug;
  }

  private setSession(res: LoginResponse, slug: string) {
    this.token = res.token!;
    this.user = res.user ?? null;
    localStorage.setItem(TOKEN_KEY, this.token!);
    const s = this.user?.tenant?.slug ?? slug.trim().toLowerCase();
    if (s) {
      this.tenantSlug = s;
      localStorage.setItem(SLUG_KEY, s);
    }
  }

  async logout(): Promise<void> {
    try {
      await api.post(`${AUTH_BASE}/logout`);
    } catch {
      // already invalid server-side
    }
    this.clear();
    goto(resolve('/login'));
  }

  /** 401 from any API call lands here via the client. */
  unauthorized(): void {
    this.clear();
    goto(resolve('/login'));
  }

  clear(): void {
    this.token = null;
    this.user = null;
    localStorage.removeItem(TOKEN_KEY);
  }

  isPublic(path: string): boolean {
    return PUBLIC_PATHS.has(path);
  }
}

export const auth = new Auth();
