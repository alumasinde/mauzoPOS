import { describe, expect, it, vi } from 'vitest';
import { createClient } from './client';
import { ApiError } from './errors';

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}

describe('api client', () => {
  it('unwraps data and sends auth + idempotency headers', async () => {
    const fetch = vi.fn(async () => jsonResponse(201, { data: { id: 'x' } }));
    const api = createClient({ baseUrl: 'http://api/', getToken: () => 'tok', fetch });

    const out = await api.post<{ id: string }>('/api/v1/t/sales', { a: 1 }, { idempotencyKey: 'k1' });

    expect(out).toEqual({ id: 'x' });
    const [url, init] = fetch.mock.calls[0] as unknown as [string, RequestInit];
    expect(url).toBe('http://api/api/v1/t/sales');
    const h = init.headers as Record<string, string>;
    expect(h['Authorization']).toBe('Bearer tok');
    expect(h['Idempotency-Key']).toBe('k1');
    expect(init.body).toBe('{"a":1}');
  });

  it('builds list query strings and skips empty values', async () => {
    const fetch = vi.fn(async () =>
      jsonResponse(200, { data: [], meta: { page: 2, per_page: 10, total: 0 } }),
    );
    const api = createClient({ baseUrl: '', fetch });
    const res = await api.list('/p', { page: 2, per_page: 10, q: '' });
    expect((fetch.mock.calls[0] as unknown as [string])[0]).toBe('/p?page=2&per_page=10');
    expect(res.meta.page).toBe(2);
  });

  it('throws ApiError with the server error envelope', async () => {
    const onUnauthorized = vi.fn();
    const fetch = vi.fn(async () => jsonResponse(401, { error: { code: 'unauthorized', message: 'Login' } }));
    const api = createClient({ baseUrl: '', fetch, onUnauthorized });

    const err = await api.get('/x').catch((e: unknown) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect((err as ApiError).code).toBe('unauthorized');
    expect(onUnauthorized).toHaveBeenCalledOnce();
  });

  it('maps fetch failures to a network ApiError', async () => {
    const fetch = vi.fn(async () => {
      throw new TypeError('failed');
    });
    const api = createClient({ baseUrl: '', fetch });
    const err = (await api.get('/x').catch((e: unknown) => e)) as ApiError;
    expect(err.isNetwork).toBe(true);
  });
});
