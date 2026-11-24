// Shift state — the till's open drawer session. Refreshed after unlock and
// on heartbeat; shift actions are online-only (soft-gated: selling is still
// allowed without one, sales stamp their window on upload).
import { desktop, type Shift, type ShiftReport } from './desktop';
import { pulse } from './pulse.svelte';

export const shift = $state({
  current: null as Shift | null,
  checked: false, // server asked at least once
  busy: false,
  report: null as ShiftReport | null,
});

export async function refreshShift(): Promise<void> {
  const d = desktop();
  if (!d || !pulse.status?.online) return;
  try {
    const r = await d.ShiftStatus();
    shift.current = r.shift;
    shift.checked = true;
  } catch {
    // Leave the cached value — a flaky poll shouldn't flap the UI.
  }
}

export async function openShift(floatAmount: string): Promise<void> {
  shift.current = await desktop()!.ShiftOpen(floatAmount);
  shift.checked = true;
}

export async function shiftCash(direction: 'in' | 'out', amount: string, reason: string) {
  await desktop()!.ShiftCash(direction, amount, reason);
}

export async function shiftDrawer() {
  await desktop()!.ShiftDrawer();
}

export async function loadXReport(): Promise<ShiftReport> {
  shift.report = await desktop()!.ShiftXReport();
  return shift.report;
}

export async function closeShift(declared: string, note: string): Promise<ShiftReport> {
  shift.report = await desktop()!.ShiftClose(declared, note);
  shift.current = null;
  return shift.report;
}

export async function printShiftReport(rep: ShiftReport): Promise<string> {
  return desktop()!.PrintShiftReport(rep);
}
