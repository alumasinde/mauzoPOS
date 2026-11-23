// Receipts (Phase 8): layouts, rendered receipts and issued copies.
// The layout format is the backend's pkg/receiptfmt Template (schema v1).

export type ReceiptBlockType =
  | 'logo'
  | 'text'
  | 'divider'
  | 'fields'
  | 'items'
  | 'totals'
  | 'payments'
  | 'tax_summary'
  | 'fiscal'
  | 'qr'
  | 'barcode'
  | 'feed';

export type ReceiptAlign = 'left' | 'center' | 'right';
export type ReceiptSize = 'normal' | 'tall' | 'wide' | 'double';

export interface ReceiptField {
  label: string;
  value: string;
}

export interface ReceiptBlock {
  type: ReceiptBlockType;
  if?: string;
  text?: string;
  align?: ReceiptAlign;
  bold?: boolean;
  size?: ReceiptSize;
  char?: string;
  fields?: ReceiptField[];
  lines?: number;
  data?: string;
  qr_size?: number;
  layout?: 'detailed' | 'compact';
  show_sku?: boolean;
  show_promo?: boolean;
  show_item_count?: boolean;
}

export interface ReceiptTemplateBody {
  version: number;
  paper: 58 | 80;
  columns?: number;
  blocks: ReceiptBlock[];
}

export interface ReceiptTemplate {
  id: string;
  name: string;
  paper_width: 58 | 80;
  version: number;
  based_on?: string;
  body: ReceiptTemplateBody;
  updated_at: string;
}

export interface ReceiptTemplateVersion {
  version: number;
  paper_width: number;
  author?: string;
  body: ReceiptTemplateBody;
  created_at: string;
}

export interface ReceiptBuiltin {
  key: string;
  name: string;
  description: string;
  body: ReceiptTemplateBody;
}

export interface NamedDoc {
  name: string;
  description: string;
}

export interface ReceiptMeta {
  builtins: ReceiptBuiltin[];
  variables: NamedDoc[];
  conditions: NamedDoc[];
  schema_version: number;
  has_logo: boolean;
}

export interface ReceiptAssignment {
  branch_id: string; // "" = all branches
  branch_name: string;
  template_id: string;
  template_name: string;
}

/** One laid-out line of a rendered receipt (compact keys from the API). */
export interface ReceiptLine {
  k: 'text' | 'logo' | 'qr' | 'barcode' | 'feed';
  t?: string;
  a?: ReceiptAlign;
  b?: boolean;
  s?: 'tall' | 'wide' | 'double';
  d?: string;
  q?: number;
  n?: number;
  bars?: string;
}

export interface ReceiptOutput {
  paper: number;
  width: number;
  lines: ReceiptLine[];
}

export interface ReceiptPreview {
  output: ReceiptOutput;
  has_logo: boolean;
}

export interface IssuedReceipt {
  id: string;
  copy_no: number;
  channel: 'pos' | 'backoffice';
  template_name?: string;
  template_version?: number;
  builtin_key?: string;
  issued_by?: string;
  created_at: string;
}

export interface SaleReceipt {
  output: ReceiptOutput;
  has_logo: boolean;
  issued: IssuedReceipt;
  copies?: IssuedReceipt[];
}

/** Full receipt data of a sale (A4 invoice). */
export interface ReceiptDoc {
  business: {
    name: string;
    address?: string;
    phone?: string;
    email?: string;
    kra_pin?: string;
    vat_registered?: boolean;
  };
  branch: { name: string; code?: string; phone?: string; location?: string };
  sale: {
    receipt_no?: string;
    ref?: string;
    sold_at: string;
    cashier?: string;
    terminal?: string;
    customer?: string;
    customer_pin?: string;
    currency?: string;
    subtotal: string;
    discount?: string;
    discount_note?: string;
    tax?: string;
    total: string;
    paid?: string;
    change?: string;
    prices_include_vat?: boolean;
  };
  items: {
    name: string;
    sku?: string;
    qty: string;
    unit_price: string;
    discount?: string;
    total: string;
    tax_rate?: string | null;
    tax?: string;
  }[];
  payments: { method: string; amount: string; tendered?: string; change?: string; reference?: string }[];
  taxes?: { rate: string; taxable: string; tax: string }[];
  fiscal?: { invoice_no?: string; cu_serial?: string; signature?: string; qr_data?: string } | null;
  footer?: string;
  extra?: Record<string, string>;
}
