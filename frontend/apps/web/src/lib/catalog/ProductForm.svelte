<script lang="ts">
  import type { Brand, Category, ProductDetail, ProductKind, SellBy, Unit } from '@mauzo/types';
  import { Button, Modal, Switch, TextField, toast } from '@mauzo/ui';
  import { ApiError } from '@mauzo/api-client';
  import RowSection from './RowSection.svelte';
  import type { AttrRow, BarcodeRow, ComponentRow, VariantRow } from './rows.ts';

  interface Props {
    open: boolean;
    product: ProductDetail | null;
    categories: Category[];
    brands: Brand[];
    units: Unit[];
    /** Active products for the composite-component picker. */
    products: { id: string; sku: string; name: string }[];
    onsaved: () => void;
    save: (body: unknown) => Promise<unknown>;
  }

  let { open = $bindable(), product, categories, brands, units, products, onsaved, save }: Props = $props();

  const KINDS: { value: ProductKind; label: string }[] = [
    { value: 'standard', label: 'Standard' },
    { value: 'service', label: 'Service' },
    { value: 'packaged', label: 'Packaged' },
    { value: 'composite', label: 'Composite (made of other products)' },
  ];

  let sku = $state('');
  let name = $state('');
  let description = $state('');
  let kind = $state<ProductKind>('standard');
  let sellBy = $state<SellBy>('unit');
  let categoryId = $state('');
  let brandId = $state('');
  let unitCode = $state('');
  let price = $state('');
  let cost = $state('');
  let vatRate = $state('');
  let deposit = $state('');
  let trackStock = $state(true);
  let variants = $state<VariantRow[]>([]);
  let barcodes = $state<BarcodeRow[]>([]);
  let components = $state<ComponentRow[]>([]);
  let attrs = $state<AttrRow[]>([]);
  let saving = $state(false);
  let errors = $state<Record<string, string>>({});

  const isService = $derived(kind === 'service');
  const isComposite = $derived(kind === 'composite');
  const weighted = $derived(sellBy === 'weight');

  // Populate the form when `product` changes (null → blank create form).
  $effect(() => {
    if (!open) return;
    const p = product;
    sku = p?.sku ?? '';
    name = p?.name ?? '';
    kind = p?.kind ?? 'standard';
    sellBy = p?.sell_by ?? 'unit';
    categoryId = p?.category_id ?? '';
    brandId = p?.brand_id ?? '';
    unitCode = p?.unit_code ?? '';
    price = p?.price ?? '';
    cost = p?.cost ?? '';
    vatRate = p?.vat_rate ?? '';
    deposit = p?.deposit ?? '';
    trackStock = p?.track_stock ?? true;
    variants = p?.variants.map((v) => ({ name: v.name, sku: v.sku ?? '', price: v.price ?? '' })) ?? [];
    barcodes =
      p?.barcodes.map((b) => ({
        code: b.code,
        variant: p.variants.find((v) => v.id === b.variant_id)?.name ?? '',
        pack_qty: b.pack_qty,
      })) ?? [];
    components = p?.components.map((c) => ({ product_id: c.product_id, qty: c.qty })) ?? [];
    attrs = Object.entries(p?.attributes ?? {}).map(([key, value]) => ({ key, value }));
    description = '';
    errors = {};
  });

  function body() {
    const attributes: Record<string, string> = {};
    for (const a of attrs) if (a.key.trim()) attributes[a.key.trim()] = a.value;
    return {
      sku: sku.trim(),
      name: name.trim(),
      description,
      kind,
      sell_by: sellBy,
      category_id: categoryId || null,
      brand_id: brandId || null,
      unit_code: unitCode,
      price,
      cost,
      vat_rate: vatRate,
      deposit,
      track_stock: trackStock,
      attributes: Object.keys(attributes).length ? attributes : undefined,
      variants: variants
        .filter((v) => v.name.trim())
        .map((v) => ({ name: v.name.trim(), sku: v.sku.trim(), price: v.price || null })),
      barcodes: barcodes
        .filter((b) => b.code.trim())
        .map((b) => ({
          code: b.code.trim(),
          variant_id: b.variant || null, // name during create — server resolves
          pack_qty: b.pack_qty || '1',
        })),
      components: isComposite
        ? components.filter((c) => c.product_id).map((c) => ({ product_id: c.product_id, qty: c.qty }))
        : [],
    };
  }

  async function submit() {
    saving = true;
    errors = {};
    try {
      await save(body());
      toast.success(product ? 'Product updated' : 'Product created');
      open = false;
      onsaved();
    } catch (e) {
      if (e instanceof ApiError && e.details) errors = e.details as Record<string, string>;
      else toast.error(e instanceof ApiError ? e.message : 'Save failed');
    } finally {
      saving = false;
    }
  }
</script>

<Modal bind:open title={product ? 'Edit product' : 'New product'} size="lg">
  <div class="grid">
    <TextField label="SKU *" bind:value={sku} error={errors.sku} />
    <TextField label="Name *" bind:value={name} error={errors.name} />
    <label class="field">
      <span>Kind</span>
      <select bind:value={kind}>
        {#each KINDS as k (k.value)}<option value={k.value}>{k.label}</option>{/each}
      </select>
    </label>
    <label class="field">
      <span>Sell by</span>
      <select bind:value={sellBy} disabled={isService}>
        <option value="unit">Unit / each</option>
        <option value="weight">Weight (price per unit)</option>
      </select>
    </label>
    <label class="field">
      <span>Category</span>
      <select bind:value={categoryId}>
        <option value="">—</option>
        {#each categories as c (c.id)}<option value={c.id}>{c.name}</option>{/each}
      </select>
    </label>
    <label class="field">
      <span>Brand</span>
      <select bind:value={brandId}>
        <option value="">—</option>
        {#each brands as b (b.id)}<option value={b.id}>{b.name}</option>{/each}
      </select>
    </label>
    <label class="field">
      <span>Unit{weighted ? ' *' : ''}</span>
      <select bind:value={unitCode}>
        <option value="">—</option>
        {#each units as u (u.id)}<option value={u.code}>{u.name} ({u.code})</option>{/each}
      </select>
      {#if errors.unit_code}<small class="err">{errors.unit_code}</small>{/if}
    </label>
    <TextField
      label={weighted ? 'Price per unit *' : 'Price *'}
      bind:value={price}
      error={errors.price}
      placeholder="0.00"
    />
    <TextField label="Cost" bind:value={cost} placeholder="0.00" />
    <TextField label="VAT rate %" bind:value={vatRate} placeholder="16" />
    <TextField label="Deposit (e.g. bottle)" bind:value={deposit} placeholder="0.00" />
    <div class="switch">
      <Switch
        checked={trackStock}
        label="Track stock"
        disabled={isService}
        onchange={(v) => (trackStock = v)}
      />
    </div>
  </div>

  <RowSection
    title="Barcodes"
    items={barcodes}
    cols="1fr 1fr 100px"
    onadd={() => (barcodes = [...barcodes, { code: '', variant: '', pack_qty: '1' }])}
    onremove={(i) => (barcodes = barcodes.filter((_, j) => j !== i))}
  >
    {#snippet row(b: BarcodeRow)}
      <input placeholder="Scan or type code" bind:value={b.code} />
      <select bind:value={b.variant}>
        <option value="">Whole product</option>
        {#each variants.filter((v) => v.name.trim()) as v (v.name)}<option value={v.name}>
            {v.name}
          </option>{/each}
      </select>
      <input class="qty" placeholder="Pack qty" bind:value={b.pack_qty} />
    {/snippet}
  </RowSection>

  <RowSection
    title="Variants"
    items={variants}
    cols="1fr 1fr 110px"
    onadd={() => (variants = [...variants, { name: '', sku: '', price: '' }])}
    onremove={(i) => (variants = variants.filter((_, j) => j !== i))}
  >
    {#snippet row(v: VariantRow)}
      <input placeholder="Name (e.g. 500ml, Red, Crate of 24)" bind:value={v.name} />
      <input placeholder="SKU" bind:value={v.sku} />
      <input class="qty" placeholder="Price (− same)" bind:value={v.price} />
    {/snippet}
  </RowSection>

  {#if isComposite}
    <RowSection
      title="Components *"
      items={components}
      cols="1fr 100px"
      onadd={() => (components = [...components, { product_id: '', qty: '1' }])}
      onremove={(i) => (components = components.filter((_, j) => j !== i))}
    >
      {#snippet row(c: ComponentRow)}
        <select bind:value={c.product_id}>
          <option value="">Choose product…</option>
          {#each products.filter((p) => p.sku !== sku) as p (p.id)}<option value={p.id}>
              {p.name} ({p.sku})
            </option>{/each}
        </select>
        <input class="qty" placeholder="Qty" bind:value={c.qty} />
      {/snippet}
    </RowSection>
  {/if}

  <RowSection
    title="Attributes"
    items={attrs}
    cols="1fr 1fr"
    onadd={() => (attrs = [...attrs, { key: '', value: '' }])}
    onremove={(i) => (attrs = attrs.filter((_, j) => j !== i))}
  >
    {#snippet row(a: AttrRow)}
      <input placeholder="Key (e.g. cut, origin)" bind:value={a.key} />
      <input placeholder="Value" bind:value={a.value} />
    {/snippet}
  </RowSection>

  {#snippet footer()}
    <Button variant="ghost" onclick={() => (open = false)}>Cancel</Button>
    <Button onclick={submit} disabled={saving}>{saving ? 'Saving…' : 'Save product'}</Button>
  {/snippet}
</Modal>

<style>
  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
    gap: 0.75rem;
  }
  .field {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    font-size: 0.85rem;
  }
  .field span {
    font-weight: 500;
    color: var(--text-muted, #667);
  }
  select,
  input {
    padding: 0.45rem 0.6rem;
    border: 1px solid var(--border, #d8dce3);
    border-radius: 6px;
    font: inherit;
    background: var(--surface, #fff);
  }
  .err {
    color: var(--danger, #c0392b);
    font-size: 0.75rem;
  }
  .switch {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.85rem;
    margin-top: 1.2rem;
  }
</style>
