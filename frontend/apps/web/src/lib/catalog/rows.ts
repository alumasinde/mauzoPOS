/** Editable row shapes for ProductForm's repeating child lists. `variant`
 *  holds the variant *name* during editing — the server resolves it to an id. */
export interface VariantRow {
  name: string;
  sku: string;
  price: string;
}

export interface BarcodeRow {
  code: string;
  variant: string; // variant name ('' = whole product)
  pack_qty: string;
}

export interface ComponentRow {
  product_id: string;
  qty: string;
}

export interface AttrRow {
  key: string;
  value: string;
}
