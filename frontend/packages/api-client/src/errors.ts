import type { ApiErrorBody } from '@mauzo/types';

/** Thrown for any non-2xx response or network failure. */
export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly details?: Record<string, unknown>;

  constructor(status: number, body: ApiErrorBody) {
    super(body.message);
    this.name = 'ApiError';
    this.status = status;
    this.code = body.code;
    this.details = body.details;
  }

  get isNetwork(): boolean {
    return this.status === 0;
  }
  get isUnauthorized(): boolean {
    return this.status === 401;
  }
  get isValidation(): boolean {
    return this.status === 422;
  }
}

export function networkError(message = 'Cannot reach the server. Check your connection.'): ApiError {
  return new ApiError(0, { code: 'network_error', message });
}
