import { goto } from '$app/navigation';
import { resolve } from '$app/paths';
import type { LoginResponse, Profile } from '@mauzo/types';
import { api } from './api';

const TOKEN_KEY = 'mauzo_platform_token';
export const AUTH_BASE = '/api/v1/platform/auth';

/** Public routes that never need a session. */
const PUBLIC_PATHS = new Set(['/login', '/forgot-password', '/reset-password']);

class Auth {
  token = $state<string | null>(null);
  user = $state<Profile | null>(null);
  /** True once the first me() check has resolved. */
  ready = $state(false);

  /** Called from +layout on mount. */
  async init(): Promise<void> {
    this.token = localStorage.getItem(TOKEN_KEY);
    if (this.token) {
      try {
        this.user = await api.get<Profile>(`${AUTH_BASE}/me`);
      } catch {
        this.clear();
      }
    }
    this.ready = true;
  }

  /** Returns the login response — when mfa_required the caller shows the code step. */
  async login(email: string, password: string): Promise<LoginResponse> {
    const res = await api.post<LoginResponse>(`${AUTH_BASE}/login`, { email, password });
    if (res?.token) this.setSession(res);
    return res ?? {};
  }

  async verifyMFA(ticket: string, code: string): Promise<void> {
    const res = await api.post<LoginResponse>(`${AUTH_BASE}/mfa/verify`, { ticket, code });
    if (res?.token) this.setSession(res);
  }

  /** Re-fetches /me into auth.user (e.g. after MFA enable/disable). */
  async refresh(): Promise<void> {
    this.user = await api.get<Profile>(`${AUTH_BASE}/me`);
  }

  private setSession(res: LoginResponse) {
    this.token = res.token!;
    this.user = res.user ?? null;
    localStorage.setItem(TOKEN_KEY, this.token!);
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

  /** UI hint: whether the signed-in user holds perm (the API enforces it). */
  can(perm: string): boolean {
    return this.user?.permissions?.includes(perm) ?? false;
  }

  isPublic(path: string): boolean {
    return PUBLIC_PATHS.has(path);
  }
}

export const auth = new Auth();
