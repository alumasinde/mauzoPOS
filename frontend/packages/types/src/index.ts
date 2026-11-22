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
