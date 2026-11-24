import { api } from '#lib/api.ts';
import type { MoneyString } from '@mauzo/types';

/** One feed transaction (server wire shape). */
export interface MpesaTxn {
  id: number;
  public_id: string;
  branch_id: number | null;
  kind: string; // stk | c2b | reversal
  status: string; // initiated | pending | success | failed | cancelled | timeout | reversal_pending | reversed
  trans_id: string | null;
  checkout_request_id: string | null;
  phone: string;
  payer_name: string;
  amount: MoneyString;
  account_ref: string;
  short_code: string;
  result_code: number | null;
  result_desc: string;
  allocated_sale_id: number | null;
  trans_time: string | null;
  created_at: string;
}

export interface MpesaEvent {
  kind: string;
  ref: string;
  detail: string;
  created_at: string;
}

export interface MpesaSummary {
  collected_today: MoneyString;
  count_today: number;
  unallocated: number;
  pending: number;
  failed_7d: number;
  reversed: number;
}

export interface MpesaSaleBrief {
  public_id: string;
  receipt_no: number;
}

export interface MpesaCandidate {
  sale_id: number;
  public_id: string;
  receipt_no: number;
  branch_id: number;
  branch_name: string;
  cashier_name: string;
  total: MoneyString;
  mpesa_amount: MoneyString;
  reference: string;
  sold_at: string;
  allocated: boolean;
}

export interface MpesaSuggestion {
  txn: MpesaTxn;
  candidate: MpesaCandidate;
  strength: string; // strong | weak
}

export interface MpesaConfig {
  env: string;
  short_code: string;
  short_code_kind: string;
  consumer_key: string;
  consumer_secret: string;
  passkey: string;
  initiator_name: string;
  security_credential: string;
  has_consumer_secret: boolean;
  has_passkey: boolean;
  has_security_credential: boolean;
  stk_callback_url: string;
  c2b_confirmation_url: string;
  c2b_validation_url: string;
  reversal_result_url: string;
  c2b_registered_at?: string;
}

export interface TxnList {
  items: MpesaTxn[];
  total: number;
  page: number;
  per_page: number;
}

export interface TxnFilters {
  status?: string;
  kind?: string;
  allocated?: string; // yes | no
  q?: string;
  from?: string;
  to?: string;
}

function qs(f: TxnFilters, page: number, per = 25): string {
  const p = new URLSearchParams();
  for (const [k, v] of Object.entries(f)) if (v) p.set(k, v);
  p.set('page', String(page));
  p.set('per_page', String(per));
  return p.toString();
}

export const mpesaApi = {
  list: (f: TxnFilters, page = 1) => api.get<TxnList>(`/api/v1/t/mpesa/transactions?${qs(f, page)}`),
  get: (id: string) =>
    api.get<{ txn: MpesaTxn; events: MpesaEvent[]; sale: MpesaSaleBrief | null }>(
      `/api/v1/t/mpesa/transactions/${id}`,
    ),
  summary: () => api.get<MpesaSummary>('/api/v1/t/mpesa/summary'),
  suggestions: () => api.get<{ items: MpesaSuggestion[] }>('/api/v1/t/mpesa/suggestions'),
  allocate: (id: string, saleID: string) =>
    api.post(`/api/v1/t/mpesa/transactions/${id}/allocate`, { sale_id: saleID }),
  reallocate: (id: string, saleID: string) =>
    api.post(`/api/v1/t/mpesa/transactions/${id}/reallocate`, { sale_id: saleID }),
  reverse: (id: string, reason: string) => api.post(`/api/v1/t/mpesa/transactions/${id}/reverse`, { reason }),
  config: () => api.get<MpesaConfig>('/api/v1/t/mpesa/config'),
  saveConfig: (c: Partial<MpesaConfig>) => api.put('/api/v1/t/mpesa/config', c),
  registerC2B: () => api.post('/api/v1/t/mpesa/config/register', {}),
  testPush: (phone: string) => api.post('/api/v1/t/mpesa/test', { phone }),
};
