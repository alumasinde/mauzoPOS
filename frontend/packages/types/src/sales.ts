// Phase 7 — sales, payments, customers, returns.

export type SaleStatus = 'completed' | 'voided' | 'returned' | 'partially_returned';
export type PayMethod = 'cash' | 'mpesa' | 'card' | 'bank' | 'credit';

export interface Sale {
  id: string;
  receipt_no: number;
  status: SaleStatus;
  source: 'pos' | 'backoffice';
  branch: string;
  terminal: string;
  cashier: string;
  customer: string;
  subtotal: string;
  discount: string;
  tax: string;
  total: string;
  paid: string;
  change_due: string;
  sold_at: string;
  voided_at?: string | null;
  void_reason?: string;
}

export interface SaleItem {
  line_no: number;
  name: string;
  sku: string;
  qty: string;
  unit_price: string;
  discount: string;
  tax: string;
  total: string;
  returned_qty: string;
  price_override: boolean;
  promotion?: string;
}

export interface SalePayment {
  method: PayMethod;
  amount: string;
  tendered?: string;
  change_due?: string;
  reference?: string;
  status: 'confirmed' | 'voided' | 'refunded';
}

export interface SaleReturn {
  id: string;
  return_no: number;
  receipt_no?: number;
  reason: string;
  refund_method: PayMethod;
  refund_amount: string;
  status: string;
  cashier?: string;
  created_at: string;
}

export interface SaleDetail extends Sale {
  items: SaleItem[];
  payments: SalePayment[];
  returns: SaleReturn[];
}

export interface SaleSummary {
  total: string;
  cnt: number;
  avg: string;
}

export interface MethodSplit {
  method: string;
  total: string;
  cnt: number;
}

export interface HeldSale {
  id: string;
  label: string;
  held_by: string;
  terminal: string;
  branch: string;
  cart: string;
  held_at: string;
}

export interface Customer {
  id: string;
  code: string;
  name: string;
  phone: string;
  email: string;
  kra_pin: string;
  credit_limit?: string | null;
  credit_balance: string;
  notes?: string | null;
  status: 'active' | 'archived';
  created_at: string;
}
