// Wire DTOs shared between the Go bridge and the till UI. Re-exported from
// desktop.ts so consumers keep a single import site.

/** One cashier's drawer session on this till. */
export interface Shift {
  id: string;
  status: 'open' | 'closed';
  float: string;
  opened_at: string;
  closed_at: string | null;
  opened_by_name: string;
  closed_by_name: string;
  terminal: string;
  branch: string;
  expected: string | null;
  declared: string | null;
  variance: string | null;
  close_note: string;
}

/** X-report (open) or Z-report (closed) data for a shift. */
export interface ShiftReport {
  shift: Shift;
  sales_count: number;
  sales_total: string;
  methods: { method: string; count: number; total: string }[];
  cash_sales: string;
  cash_refunded: string;
  cash_in: string;
  cash_out: string;
  expected: string;
  movements: {
    id: string;
    kind: 'cash_in' | 'cash_out' | 'drawer_open';
    amount: string;
    reason: string;
    created_at: string;
    created_by_name: string;
  }[];
}

/** An offline sale the server refused on upload. */
export interface FailedSale {
  id: string;
  payload: string;
  cashier_id: string;
  created_at: string;
  error: string;
}
