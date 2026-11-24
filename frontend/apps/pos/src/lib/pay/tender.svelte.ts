// Tender state for one checkout: the payment lines taken so far, what is
// still owed and the change due. Amounts are kept in cents (integers) so
// splits never drift by a rounding error.

export type Method = 'cash' | 'mpesa' | 'card' | 'bank' | 'credit';

export interface Line {
  method: Method;
  /** Applied to the sale, in cents. */
  applied: number;
  /** What the customer handed over / sent (cash, M-Pesa), in cents. */
  tendered?: number;
  reference?: string;
  /** Display detail: payer name, card digits… */
  note?: string;
}

export const LABELS: Record<Method, string> = {
  cash: 'Cash',
  mpesa: 'M-Pesa',
  card: 'Card',
  bank: 'Bank',
  credit: 'On account',
};

export const toCents = (v: string | number): number => Math.round((Number(v) || 0) * 100);
export const fromCents = (c: number): string => (c / 100).toFixed(2);

export class Tender {
  /** Sale total in cents. */
  total = $state(0);
  lines = $state<Line[]>([]);

  constructor(totalCents: number) {
    this.total = totalCents;
  }

  paid = $derived(this.lines.reduce((n, l) => n + l.applied, 0));
  remaining = $derived(Math.max(0, this.total - this.paid));
  /** Cash handed back: overpaid cash plus M-Pesa sent above the bill. */
  change = $derived(this.lines.reduce((n, l) => n + Math.max(0, (l.tendered ?? l.applied) - l.applied), 0));
  settled = $derived(this.lines.length > 0 && this.paid === this.total);

  /**
   * Takes a payment. received is what the customer gave; at most the
   * remaining balance is applied and any excess becomes change (cash and
   * M-Pesa only — card/bank/account lines are charged exactly).
   */
  take(method: Method, received: number, reference = '', note = ''): Line | null {
    if (received <= 0 || this.remaining <= 0) return null;
    const applied = Math.min(received, this.remaining);
    const line: Line = { method, applied, reference: reference || undefined, note: note || undefined };
    if (method === 'cash' || method === 'mpesa') line.tendered = received;
    this.lines = [...this.lines, line];
    return line;
  }

  remove(i: number): void {
    this.lines = this.lines.filter((_, j) => j !== i);
  }

  /** Uses an M-Pesa code at most once per sale. */
  hasReference(ref: string): boolean {
    return this.lines.some((l) => l.reference === ref);
  }

  /** The payments array for the Pay binding. */
  payload(): { method: string; amount: string; tendered?: string; reference?: string }[] {
    return this.lines.map((l) => ({
      method: l.method,
      amount: fromCents(l.applied),
      ...(l.tendered !== undefined ? { tendered: fromCents(l.tendered) } : {}),
      ...(l.reference ? { reference: l.reference } : {}),
    }));
  }
}

/** Notes and coins a Kenyan customer is likely to hand over for amount. */
export function quickCash(dueCents: number): number[] {
  const due = Math.ceil(dueCents / 100);
  const out: number[] = [due];
  for (const step of [50, 100, 500, 1000]) out.push(Math.ceil(due / step) * step);
  for (const note of [50, 100, 200, 500, 1000]) if (note > due) out.push(note);
  return out
    .filter((v, i) => v >= due && out.indexOf(v) === i)
    .sort((a, b) => a - b)
    .slice(0, 6)
    .map((v) => v * 100);
}
