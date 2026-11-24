import { desktop, type PricedLine, type Product, type Quote, type ScannedItem } from './desktop';

// One cart row: a product/variant at a quantity with optional overrides.
export interface CartRow {
  product_id: number;
  variant_id: number;
  category_id: number;
  name: string;
  sku: string;
  unit: string;
  sell_by: string;
  qty: number;
  price_override: string; // '' = resolve normally
  line_discount: string; // absolute amount off the line
  priced?: PricedLine; // filled by requote()
}

const EMPTY: Quote = { subtotal: '0.00', discount: '0.00', vat: '0.00', total: '0.00', lines: [] };

function newCart() {
  let rows = $state<CartRow[]>([]);
  let totals = $state<Quote>(EMPTY);
  let selected = $state(0);
  // One-time supervisor grant + chosen customer, consumed by the next Pay.
  let grant = $state('');
  let customerID = $state('');
  let customerName = $state('');
  let customerPhone = $state('');
  // M-Pesa code picked from the collection feed — seeds the mpesa row's
  // reference in the next PayModal.
  let mpesaRef = $state('');
  const bridge = desktop();

  function sameLine(r: CartRow, product_id: number, variant_id: number) {
    return r.product_id === product_id && r.variant_id === variant_id && !r.price_override;
  }

  async function requote() {
    totals = EMPTY;
    if (!bridge || rows.length === 0) return;
    const q = await bridge.Quote(
      rows.map((r) => ({
        product_id: r.product_id,
        variant_id: r.variant_id,
        category_id: r.category_id,
        qty: String(r.qty),
      })),
    );
    totals = q;
    q.lines.forEach((pl, i) => {
      if (rows[i]) rows[i].priced = pl;
    });
  }

  return {
    get rows() {
      return rows;
    },
    get totals() {
      return totals;
    },
    get selected() {
      return selected;
    },
    get count() {
      return rows.length;
    },
    get grant() {
      return grant;
    },
    set grant(v: string) {
      grant = v;
    },
    get customerID() {
      return customerID;
    },
    get customerName() {
      return customerName;
    },
    /** Phone of the attached customer — prefills the M-Pesa prompt. */
    get customerPhone() {
      return customerPhone;
    },
    get mpesaRef() {
      return mpesaRef;
    },
    set mpesaRef(v: string) {
      mpesaRef = v;
    },
    setCustomer(id: string, name: string, phone = '') {
      customerID = id;
      customerName = name;
      customerPhone = phone;
    },

    select(i: number) {
      selected = i;
    },

    async add(item: ScannedItem) {
      const p = item.product;
      const qty = Number(item.qty) || 1;
      const existing = rows.findIndex((r) => sameLine(r, p.id, p.variant_id));
      if (existing >= 0 && !item.price_override && rows[existing]) {
        rows[existing].qty += qty;
        selected = existing;
      } else {
        rows = [
          ...rows,
          {
            product_id: p.id,
            variant_id: p.variant_id,
            category_id: p.category_id,
            name: p.variant_name ? `${p.name} · ${p.variant_name}` : p.name,
            sku: p.sku,
            unit: p.unit_code,
            sell_by: p.sell_by,
            qty,
            price_override: item.price_override,
            line_discount: '0',
          },
        ];
        selected = rows.length - 1;
      }
      await requote();
    },

    async addProduct(p: Product) {
      await this.add({ product: p, qty: '1', price_override: '', unknown: false });
    },

    async setQty(i: number, qty: number) {
      if (!rows[i]) return;
      if (qty <= 0) return this.remove(i);
      rows[i].qty = qty;
      await requote();
    },

    async bump(i: number, delta: number) {
      if (!rows[i]) return;
      const next = rows[i].qty + delta;
      await this.setQty(i, next);
    },

    async remove(i: number) {
      rows = rows.filter((_, x) => x !== i);
      selected = Math.min(selected, rows.length - 1);
      await requote();
    },

    async setDiscount(i: number, amount: string) {
      if (!rows[i]) return;
      rows[i].line_discount = amount;
      await requote();
    },

    async clear() {
      rows = [];
      totals = EMPTY;
      selected = 0;
      grant = '';
      customerID = '';
      customerName = '';
      customerPhone = '';
      mpesaRef = '';
    },

    /** Serialise for hold/sale. */
    payload(): string {
      return JSON.stringify(
        rows.map((r) => ({
          product_id: r.product_id,
          variant_id: r.variant_id,
          category_id: r.category_id,
          name: r.name,
          sku: r.sku,
          unit: r.unit,
          qty: r.qty,
          price_override: r.price_override,
          line_discount: r.line_discount,
        })),
      );
    },

    async restore(cartJSON: string) {
      try {
        const parsed = JSON.parse(cartJSON) as CartRow[];
        rows = parsed.map((r) => ({
          ...r,
          line_discount: r.line_discount ?? '0',
          price_override: r.price_override ?? '',
        }));
        selected = rows.length - 1;
        await requote();
      } catch {
        rows = [];
      }
    },
  };
}

export const cart = newCart();
