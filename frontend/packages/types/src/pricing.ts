// Phase 5 — pricing & promotions types (mirror backend DTOs).

export interface PriceList {
  id: string;
  name: string;
  branch_id?: string;
  customer_group?: string | null;
  priority: number;
  is_active: boolean;
  valid_from?: string | null;
  valid_to?: string | null;
  items?: PriceListItem[];
  created_at: string;
}

export interface PriceListItem {
  product_id: string;
  product_name: string;
  variant_id?: string;
  min_qty: string;
  price: string;
}

export interface PromotionConfig {
  percent?: string;
  amount?: string;
  buy?: string;
  get?: string;
  get_percent?: string;
}

export interface PromotionTarget {
  product_id?: string;
  category_id?: string;
}

export interface Promotion {
  id: string;
  name: string;
  kind: 'percent' | 'amount' | 'bxgy';
  config: PromotionConfig;
  starts_at?: string | null;
  ends_at?: string | null;
  days_mask?: number | null;
  priority: number;
  is_active: boolean;
  targets: PromotionTarget[];
  created_at: string;
}

export interface QuoteLine {
  product_id: string;
  variant_id?: string;
  qty: string;
}

export interface QuotedLine {
  product_id: string;
  variant_id?: string;
  product_name: string;
  qty: string;
  unit_price: string;
  source: 'base' | 'price_list';
  price_list?: string;
  discount: string;
  promotion?: string;
  final_unit_price: string;
  line_total: string;
}

export interface QuoteResponse {
  subtotal: string;
  discount: string;
  total: string;
  lines: QuotedLine[];
}

export interface PriceHistoryRow {
  product_name: string;
  source: string;
  ref_id?: string | null;
  old_price?: string | null;
  new_price: string;
  created_at: string;
}
