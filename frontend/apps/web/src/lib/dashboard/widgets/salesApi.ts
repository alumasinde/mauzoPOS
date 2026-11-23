import { api } from '#lib/api.ts';
import type { Period } from '../period';

export interface SaleSummary {
  total: string;
  cnt: number;
  avg: string;
}

export interface DailyPoint {
  day: string;
  total: string;
  cnt: number;
}

export interface MethodTotal {
  method: string;
  total: string;
  cnt: number;
  /** Joined from the method registry by fetchMethods. */
  label?: string;
}

export interface MethodSpec {
  label: string;
  requires_reference: boolean;
  requires_customer: boolean;
  tendered: boolean;
}

interface MethodsResponse {
  methods: Record<string, MethodSpec>;
  totals: MethodTotal[];
}

/** Inclusive [from, to] date strings for a period, plus the previous window. */
export function windowFor(period: Period): { from: string; to: string; prevFrom: string; prevTo: string } {
  const today = new Date();
  const fmt = (d: Date) => d.toISOString().slice(0, 10);
  const days = period === 'today' ? 1 : period === '7d' ? 7 : period === '30d' ? 30 : 90;
  const to = new Date(today);
  const from = new Date(today);
  from.setDate(from.getDate() - days + 1);
  const prevTo = new Date(from);
  prevTo.setDate(prevTo.getDate() - 1);
  const prevFrom = new Date(prevTo);
  prevFrom.setDate(prevFrom.getDate() - days + 1);
  return { from: fmt(from), to: fmt(to), prevFrom: fmt(prevFrom), prevTo: fmt(prevTo) };
}

function qs(period: Period, branchId: string | null, prev = false): string {
  const w = windowFor(period);
  const p = new URLSearchParams();
  p.set('from', prev ? w.prevFrom : w.from);
  p.set('to', prev ? w.prevTo : w.to);
  if (branchId) p.set('branch_id', branchId);
  return p.toString();
}

export function fetchSummary(period: Period, branchId: string | null, prev = false): Promise<SaleSummary> {
  return api.get<SaleSummary>(`/api/v1/t/sales/summary?${qs(period, branchId, prev)}`);
}

export function fetchDaily(period: Period, branchId: string | null): Promise<DailyPoint[]> {
  return api.get<DailyPoint[]>(`/api/v1/t/sales/daily?${qs(period, branchId)}`);
}

/** Per-method totals joined with the registry label. */
export async function fetchMethods(
  period: Period,
  branchId: string | null,
  prev = false,
): Promise<MethodTotal[]> {
  const res = await api.get<MethodsResponse>(`/api/v1/t/sales/methods?${qs(period, branchId, prev)}`);
  return (res.totals ?? []).map((t) => ({
    ...t,
    label: res.methods?.[t.method]?.label ?? t.method,
  }));
}

/** % change vs the previous window; null when there's no baseline. */
export function delta(current: number, previous: number): number | null {
  if (previous <= 0) return current > 0 ? 100 : null;
  return Math.round(((current - previous) / previous) * 100);
}
