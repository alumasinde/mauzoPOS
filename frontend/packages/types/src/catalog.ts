// ---- Catalog (Phase 4) -----------------------------------------------------

export interface Category {
  id: string;
  parent_id: string | null;
  name: string;
  sort: number;
  created_at: string;
}

export interface Brand {
  id: string;
  name: string;
}

export type UnitKind = 'count' | 'mass' | 'volume' | 'length' | 'time';

export interface Unit {
  id: string;
  code: string;
  name: string;
  kind: UnitKind;
  base_code: string | null;
  factor: string | null;
  built_in: boolean;
}

export type ProductKind = 'standard' | 'service' | 'packaged' | 'composite';
export type SellBy = 'unit' | 'weight';

export interface ProductVariant {
  id: string;
  sku: string | null;
  name: string;
  price: string | null;
  attributes: Record<string, string> | null;
}

export interface ProductBarcode {
  code: string;
  variant_id: string | null;
  pack_qty: string;
}

export interface ProductComponent {
  product_id: string;
  sku: string;
  name: string;
  qty: string;
}

export interface ProductImage {
  id: number;
  url: string;
  sort: number;
}

export interface Product {
  id: string;
  sku: string;
  name: string;
  kind: ProductKind;
  category_id: string | null;
  brand_id: string | null;
  unit_code: string | null;
  sell_by: SellBy;
  price: string;
  cost: string | null;
  vat_rate: string | null;
  deposit: string | null;
  track_stock: boolean;
  attributes: Record<string, string> | null;
  status: string;
  category_name: string | null;
  brand_name: string | null;
  updated_at: string;
}

export interface ProductDetail extends Product {
  variants: ProductVariant[];
  barcodes: ProductBarcode[];
  components: ProductComponent[];
  images: ProductImage[];
}

export interface ImportReport {
  created: number;
  updated: number;
  errors: { row: number; message: string }[];
}

export interface Label {
  product_id: string;
  name: string;
  price: string;
  barcode: string;
  pattern: boolean[];
}
