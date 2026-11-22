import type { ApiErrorBody, Envelope, Paginated, PageQuery } from '@mauzo/types';
import { ApiError, networkError } from './errors';

export interface ClientOptions {
  /** e.g. "" (same origin, dev proxy) or "https://api.example.com" */
  baseUrl: string;
  /** Returns the current bearer token, if any. */
  getToken?: () => string | null | undefined;
  /** Called on 401 so the app can redirect to login. */
  onUnauthorized?: () => void;
  timeoutMs?: number;
  fetch?: typeof fetch;
}

export interface RequestOptions {
  query?: Record<string, string | number | boolean | undefined>;
  body?: unknown;
  /** Required by the API for money/stock mutations. */
  idempotencyKey?: string;
  signal?: AbortSignal;
}

export type Client = ReturnType<typeof createClient>;

export function createClient(opts: ClientOptions) {
  const doFetch = opts.fetch ?? globalThis.fetch.bind(globalThis);
  const timeoutMs = opts.timeoutMs ?? 20_000;

  async function request<T>(method: string, path: string, ro: RequestOptions = {}): Promise<T> {
    const url = buildUrl(opts.baseUrl, path, ro.query);
    const headers: Record<string, string> = { Accept: 'application/json' };
    if (ro.body !== undefined) headers['Content-Type'] = 'application/json';
    if (ro.idempotencyKey) headers['Idempotency-Key'] = ro.idempotencyKey;
    const token = opts.getToken?.();
    if (token) headers['Authorization'] = `Bearer ${token}`;

    const timeout = AbortSignal.timeout(timeoutMs);
    const signal = ro.signal ? AbortSignal.any([ro.signal, timeout]) : timeout;

    let res: Response;
    try {
      res = await doFetch(url, {
        method,
        headers,
        body: ro.body === undefined ? undefined : JSON.stringify(ro.body),
        signal,
      });
    } catch (err) {
      if (ro.signal?.aborted) throw err;
      throw networkError();
    }

    if (res.status === 204) return undefined as T;
    const payload: unknown = await res.json().catch(() => null);

    if (!res.ok) {
      if (res.status === 401) opts.onUnauthorized?.();
      const body = (payload as { error?: ApiErrorBody } | null)?.error ?? {
        code: 'http_' + res.status,
        message: res.statusText || 'Request failed',
      };
      throw new ApiError(res.status, body);
    }
    return payload as T;
  }

  return {
    request,
    get: <T>(path: string, ro?: RequestOptions) => request<Envelope<T>>('GET', path, ro).then((e) => e.data),
    list: <T>(path: string, q?: PageQuery & RequestOptions['query'], ro?: RequestOptions) =>
      request<Paginated<T>>('GET', path, { ...ro, query: q }),
    post: <T>(path: string, body?: unknown, ro?: RequestOptions) =>
      request<Envelope<T>>('POST', path, { ...ro, body }).then((e) => e?.data),
    put: <T>(path: string, body?: unknown, ro?: RequestOptions) =>
      request<Envelope<T>>('PUT', path, { ...ro, body }).then((e) => e?.data),
    patch: <T>(path: string, body?: unknown, ro?: RequestOptions) =>
      request<Envelope<T>>('PATCH', path, { ...ro, body }).then((e) => e?.data),
    del: (path: string, ro?: RequestOptions) => request<void>('DELETE', path, ro),
  };
}

function buildUrl(base: string, path: string, query?: RequestOptions['query']): string {
  const url = base.replace(/\/$/, '') + path;
  if (!query) return url;
  const qs = new URLSearchParams();
  for (const [k, v] of Object.entries(query)) {
    if (v !== undefined && v !== '') qs.set(k, String(v));
  }
  const s = qs.toString();
  return s ? `${url}?${s}` : url;
}

/** Generates an idempotency key for a single user action. */
export function newIdempotencyKey(): string {
  return crypto.randomUUID();
}
