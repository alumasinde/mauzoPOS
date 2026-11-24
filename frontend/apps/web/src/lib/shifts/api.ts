import { api } from '#lib/api.ts';
import type { MoneyString } from '@mauzo/types';

/** One drawer session (server wire shape). */
export interface Shift {
  id: string;
  status: 'open' | 'closed';
  float: MoneyString;
  opened_at: string;
  closed_at: string | null;
  opened_by_name: string;
  closed_by_name: string;
  terminal: string;
  branch: string;
  expected: MoneyString | null;
  declared: MoneyString | null;
  variance: MoneyString | null;
  close_note: string;
  variance_status: 'pending' | 'approved' | 'flagged';
  variance_note: string;
  reviewed_at: string | null;
}

export interface ShiftMovement {
  id: string;
  kind: 'cash_in' | 'cash_out' | 'drawer_open';
  amount: MoneyString;
  reason: string;
  created_at: string;
  created_by_name: string;
}

export interface MethodTotal {
  method: string;
  count: number;
  total: MoneyString;
}

/** X-report (open) / Z-report (closed) payload. */
export interface ShiftReport {
  shift: Shift;
  sales_count: number;
  sales_total: MoneyString;
  methods: MethodTotal[];
  cash_sales: MoneyString;
  cash_refunded: MoneyString;
  cash_in: MoneyString;
  cash_out: MoneyString;
  expected: MoneyString;
  movements: ShiftMovement[];
}

export interface ShiftSummary {
  open_count: number;
  expected_open: MoneyString;
  variance_today: MoneyString;
  closed_today: number;
}

export interface ShiftList {
  items: Shift[];
  total: number;
  page: number;
  per_page: number;
}

export interface ShiftFilters {
  status?: string;
  branch_id?: string;
  since?: string;
  until?: string;
}

function qs(f: ShiftFilters, page: number, per = 25): string {
  const p = new URLSearchParams();
  for (const [k, v] of Object.entries(f)) if (v) p.set(k, v);
  p.set('page', String(page));
  p.set('per_page', String(per));
  return p.toString();
}

export const shiftsApi = {
  list: (f: ShiftFilters, page = 1) => api.get<ShiftList>(`/api/v1/t/shifts?${qs(f, page)}`),
  report: (id: string) => api.get<ShiftReport>(`/api/v1/t/shifts/${id}`),
  summary: () => api.get<ShiftSummary>('/api/v1/t/shifts/summary'),
  forceClose: (id: string, note: string) => api.post(`/api/v1/t/shifts/${id}/force-close`, { note }),
  review: (id: string, status: 'approved' | 'flagged' | 'pending', note: string) =>
    api.post(`/api/v1/t/shifts/${id}/review`, { status, note }),
};
