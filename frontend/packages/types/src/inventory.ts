// Phase 5 — inventory & stock types (mirror backend DTOs).

export interface StockLocation {
  id: string;
  name: string;
  kind: string;
  is_default: boolean;
}

export interface StockBalance {
  product_id: string;
  product_name: string;
  sku: string;
  variant_id?: string;
  variant_name?: string;
  location_id: string;
  location_name: string;
  branch_id: string;
  branch_name: string;
  batch_id?: number;
  qty: string;
  avg_cost: string;
}

export interface StockMovement {
  id: number;
  product: string;
  sku: string;
  variant_name?: string;
  location_name: string;
  branch_name: string;
  qty: string;
  kind: string;
  ref_type?: string;
  ref_id?: string;
  reason?: string;
  note?: string;
  created_at: string;
}

export interface DocItem {
  id?: string;
  product_id: string;
  product_name: string;
  variant_id?: string;
  variant_name?: string;
  location_id?: string;
  location_name?: string;
  qty?: string;
  qty_sent?: string;
  qty_received?: string | null;
  expected_qty?: string;
  counted_qty?: string | null;
}

export interface Adjustment {
  id: string;
  branch_id: string;
  branch_name: string;
  status: 'draft' | 'approved' | 'rejected' | 'posted';
  reason: string;
  note?: string | null;
  items: DocItem[];
  posted_at?: string | null;
  created_at: string;
}

export interface Transfer {
  id: string;
  from_branch: string;
  from_name: string;
  to_branch: string;
  to_name: string;
  status: 'draft' | 'sent' | 'received' | 'cancelled';
  note?: string | null;
  items: DocItem[];
  sent_at?: string | null;
  received_at?: string | null;
  created_at: string;
}

export interface StockCount {
  id: string;
  branch_id: string;
  branch_name: string;
  status: 'open' | 'posted' | 'cancelled';
  location_id?: string | null;
  note?: string | null;
  items: DocItem[];
  posted_at?: string | null;
  created_at: string;
}

export interface ProductBatch {
  id: number;
  batch_no: string;
  expiry_date?: string | null;
  received_at: string;
}

export interface ExpiringRow {
  batch_no: string;
  product_name: string;
  branch_name: string;
  qty: string;
  expiry_date?: string | null;
}

export interface ReorderRow {
  product_id: string;
  product_name: string;
  sku: string;
  variant_id?: string;
  variant_name?: string;
  min_qty: string;
  target_qty: string;
  qty: string;
}

export interface ValuationRow {
  branch_id: number;
  branch_name: string;
  sku_count: number;
  units: string;
  value: string;
}

export interface StockLineRequest {
  product_id: string;
  variant_id?: string;
  location_id?: string;
  batch_no?: string;
  expiry?: string;
  qty: string;
  unit_cost?: string;
}
