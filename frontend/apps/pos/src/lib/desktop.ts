// Bridge to Go methods bound by Wails (backend/desktop/bindings).
// The POS talks to the central API from Go, never directly from the WebView,
// so terminal credentials never reach JavaScript.

export interface TerminalInfo {
  version: string;
  api_base_url: string;
  activated: boolean;
  terminal: string;
  branch: string;
  tenant: string;
  business_type: string;
}

export interface ServerStatus {
  reachable: boolean;
  status: string;
  version: string;
  latency_ms: number;
  error?: string;
}

export interface Status {
  online: boolean;
  last_sync: string;
  printer: string; // none | ready | not_configured
  pending_sales: number;
}

export interface Cashier {
  id: string;
  name: string;
  role_name: string;
}

export interface Session {
  cashier_id: string;
  name: string;
  role_name: string;
  permissions: string[];
  since: string;
}

export interface Product {
  id: number;
  public_id: string;
  sku: string;
  name: string;
  kind: string;
  category_id: number;
  unit_code: string;
  sell_by: string; // unit | weight
  price: string;
  vat_rate: string | null;
  track_stock: boolean;
  variant_id: number;
  variant_name: string;
}

export interface Category {
  id: number;
  public_id: string;
  name: string;
  sort: number;
  count: number;
}

export interface ScannedItem {
  product: Product;
  qty: string;
  price_override: string;
  unknown: boolean;
}

export interface QuoteLine {
  product_id: number;
  variant_id: number;
  category_id: number;
  qty: string;
}

export interface PricedLine extends QuoteLine {
  unit_price: string;
  discount: string;
  total: string;
  promo_name?: string;
  source: string;
}

export interface Quote {
  subtotal: string;
  discount: string;
  vat: string;
  total: string;
  lines: PricedLine[];
}

export interface Held {
  id: string;
  label: string;
  cart: string;
  held_by: string;
  held_at: string;
  source: string; // server | local
}

export interface SyncResult {
  ok: boolean;
  at: string;
  products: number;
  categories: number;
  cashiers: number;
  error?: string;
}

export interface ActivateResult {
  token: string;
  terminal: {
    id: string;
    name: string;
    branch: string;
    tenant: string;
    slug: string;
    business_type: string;
  };
}

export interface PayResult {
  sale_id: string;
  receipt_no: number;
  total: string;
  change_due: string;
  offline: boolean;
}

export interface AuthorizeResult {
  name: string;
  grant: string;
}

export interface Customer {
  id: string;
  code: string;
  name: string;
  phone: string;
  credit_limit: string;
  credit_balance: string;
}

export interface POSLayout {
  business_type: string;
  grid_columns: number;
  weighted_items: boolean;
  tables_first: boolean;
  age_check: boolean;
  expiry_tracking: boolean;
  show_images: boolean;
  numpad: boolean;
}

export interface TerminalSettings {
  device_name: string;
  api_url: string;
  printer_name: string;
  auto_lock_seconds: number;
  business_type: string;
  sync_cursor: string;
}

export interface PriceCheckResult {
  unknown?: boolean;
  code?: string;
  name?: string;
  sku?: string;
  price?: string;
  unit?: string;
  sell_by?: string;
}

interface DesktopApp {
  Info(): Promise<TerminalInfo>;
  CheckServer(): Promise<ServerStatus>;
  Status(): Promise<Status>;
  Activate(req: { api_url: string; code: string; device_name: string }): Promise<ActivateResult>;
  Sync(): Promise<SyncResult>;
  LastSync(): Promise<string>;
  Cashiers(): Promise<Cashier[]>;
  Login(cashierID: string, pin: string): Promise<Session>;
  Logout(): Promise<void>;
  CurrentSession(): Promise<Session | null>;
  Authorize(permission: string, pin: string): Promise<AuthorizeResult>;
  Can(permission: string): Promise<boolean>;
  Categories(): Promise<Category[]>;
  Grid(categoryID: number): Promise<Product[]>;
  SearchProducts(q: string): Promise<Product[]>;
  Scan(code: string): Promise<ScannedItem>;
  PriceCheck(code: string): Promise<PriceCheckResult>;
  Quote(lines: QuoteLine[]): Promise<Quote>;
  Hold(label: string, cartJSON: string): Promise<Held>;
  HeldList(): Promise<Held[]>;
  RecallHeld(id: string): Promise<Held>;
  CancelHeld(id: string): Promise<void>;
  Pay(
    paymentsJSON: string,
    cartJSON: string,
    orderDiscount: string,
    customerID: string,
    grant: string,
    note: string,
  ): Promise<PayResult>;
  SyncPending(): Promise<number>;
  PendingSales(): Promise<number>;
  Heartbeat(): Promise<boolean>;
  SaleDetail(ref: string): Promise<unknown>;
  Customers(search: string): Promise<Customer[]>;
  GetSettings(): Promise<TerminalSettings>;
  Layout(): Promise<POSLayout>;
  SaveSettings(s: TerminalSettings): Promise<void>;
}

declare global {
  interface Window {
    go?: { bindings?: { App?: DesktopApp } };
  }
}

/** Returns the Go bridge, or null when running in a plain browser (vite dev). */
export function desktop(): DesktopApp | null {
  return window.go?.bindings?.App ?? null;
}
