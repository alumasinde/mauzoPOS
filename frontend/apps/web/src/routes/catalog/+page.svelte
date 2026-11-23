<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { resolve } from '$app/paths';
  import { ApiError } from '@mauzo/api-client';
  import type { Brand, Category, Product, ProductDetail, Unit } from '@mauzo/types';
  import { Badge, Button, DataTable, Pager, TextField, toast } from '@mauzo/ui';
  import { api } from '#lib/api.ts';
  import { auth } from '#lib/auth.svelte.ts';
  import ProductForm from '#lib/catalog/ProductForm.svelte';
  import ProductActions from '#lib/catalog/ProductActions.svelte';
  import { exportProducts, importProducts } from '#lib/catalog/transfer.ts';

  const PER_PAGE = 25;
  let rows = $state<Product[]>([]);
  let categories = $state<Category[]>([]);
  let brands = $state<Brand[]>([]);
  let units = $state<Unit[]>([]);
  let allProducts = $state<{ id: string; sku: string; name: string }[]>([]);
  let loading = $state(true);
  let q = $state('');
  let categoryId = $state('');
  let showArchived = $state(false);
  let pageNum = $state(1);
  let total = $state(0);

  let formOpen = $state(false);
  let target = $state<ProductDetail | null>(null);
  let busy = $state<string | null>(null);
  let importInput = $state<HTMLInputElement>();

  const canManage = $derived(auth.can('catalog.product.create'));
  const canDelete = $derived(auth.can('catalog.product.delete'));

  const columns = [
    { key: 'sku', label: 'SKU', width: '110px' },
    { key: 'name', label: 'Product' },
    { key: 'category_name', label: 'Category', width: '130px' },
    { key: 'kind', label: 'Kind', width: '90px' },
    { key: 'sell', label: 'Sell', width: '80px' },
    { key: 'price', label: 'Price', width: '90px', align: 'right' as const },
    { key: 'actions', label: '', width: '220px', align: 'right' as const },
  ];

  async function load() {
    loading = true;
    try {
      const res = await api.list<Product>('/api/v1/t/catalog/products', {
        page: pageNum,
        per_page: PER_PAGE,
        q,
        category_id: categoryId,
        status: showArchived ? 'archived' : 'active',
      });
      rows = res.data;
      total = res.meta.total;
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Failed to load products');
    } finally {
      loading = false;
    }
  }

  async function loadLookups() {
    try {
      const [cats, bds, us, prods] = await Promise.all([
        api.get<Category[]>('/api/v1/t/catalog/categories'),
        api.get<Brand[]>('/api/v1/t/catalog/brands'),
        api.get<Unit[]>('/api/v1/t/catalog/units'),
        api.list<Product>('/api/v1/t/catalog/products', { per_page: 200 }),
      ]);
      categories = cats;
      brands = bds;
      units = us;
      allProducts = prods.data.map((p) => ({ id: p.id, sku: p.sku, name: p.name }));
    } catch {
      /* lookups are best-effort */
    }
  }

  async function edit(p: Product | null) {
    if (!p) {
      target = null;
      formOpen = true;
      return;
    }
    busy = p.id;
    try {
      target = await api.get<ProductDetail>(`/api/v1/t/catalog/products/${p.id}`);
      formOpen = true;
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Failed to load product');
    } finally {
      busy = null;
    }
  }

  async function archive(p: Product) {
    if (!confirm(`Archive ${p.name}? It will no longer be sellable.`)) return;
    busy = p.id;
    try {
      await api.del(`/api/v1/t/catalog/products/${p.id}`);
      toast.success('Product archived');
      load();
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Archive failed');
    } finally {
      busy = null;
    }
  }

  async function saveProduct(body: unknown) {
    if (target) return api.patch(`/api/v1/t/catalog/products/${target.id}`, body);
    return api.post('/api/v1/t/catalog/products', body);
  }

  async function doImport(ev: Event) {
    const file = (ev.target as HTMLInputElement).files?.[0];
    if (!file) return;
    try {
      const report = await importProducts(file);
      toast.success(`Imported: ${report.created} created, ${report.updated} updated`);
      for (const err of report.errors.slice(0, 5)) toast.error(`Row ${err.row}: ${err.message}`);
      load();
      loadLookups();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Import failed');
    } finally {
      if (importInput) importInput.value = '';
    }
  }

  async function doExport(format: 'csv' | 'xlsx' = 'csv') {
    try {
      await exportProducts(format);
    } catch {
      toast.error('Export failed');
    }
  }

  function printLabels() {
    const ids = rows
      .slice(0, 20)
      .map((p) => p.id)
      .join(',');
    goto(resolve(`/catalog/labels?products=${ids}`));
  }

  onMount(() => {
    load();
    loadLookups();
  });
</script>

<svelte:head><title>Products — Mauzo</title></svelte:head>

<div class="head">
  <div>
    <h1>Products</h1>
    <p class="sub">{total} products</p>
  </div>
  <div class="actions">
    {#if canManage}
      <Button variant="ghost" onclick={() => doExport('csv')}>Export CSV</Button>
      <Button variant="ghost" onclick={() => doExport('xlsx')}>Export XLSX</Button>
      <Button variant="ghost" onclick={() => importInput?.click()}>Import</Button>
      <input bind:this={importInput} type="file" accept=".csv,.xlsx" hidden onchange={doImport} />
      <Button variant="ghost" onclick={printLabels}>Print labels</Button>
      <Button onclick={() => edit(null)}>+ New product</Button>
    {/if}
  </div>
</div>

<div class="filters">
  <div class="q">
    <TextField
      placeholder="Search name or SKU…"
      bind:value={q}
      oninput={() => {
        pageNum = 1;
        load();
      }}
    />
  </div>
  <select
    bind:value={categoryId}
    onchange={() => {
      pageNum = 1;
      load();
    }}
  >
    <option value="">All categories</option>
    {#each categories as c (c.id)}<option value={c.id}>{c.name}</option>{/each}
  </select>
  <label class="chk">
    <input type="checkbox" bind:checked={showArchived} onchange={load} /> Show archived
  </label>
</div>

<DataTable
  {columns}
  {rows}
  rowKey={(r) => r.id}
  {loading}
  empty="No products yet — import a CSV or create one."
  onrowclick={canManage ? edit : undefined}
>
  {#snippet cell(row, col)}
    {#if col.key === 'kind'}
      <Badge tone={row.kind === 'composite' ? 'info' : 'neutral'}>{row.kind}</Badge>
    {:else if col.key === 'sell'}
      {row.sell_by === 'weight' ? `per ${row.unit_code ?? 'unit'}` : 'unit'}
    {:else if col.key === 'price'}
      KES {row.price}
    {:else if col.key === 'category_name'}
      {row.category_name ?? '—'}
    {:else if col.key === 'actions'}
      <ProductActions {row} {canManage} {canDelete} {busy} onedit={edit} onarchive={archive} />
    {:else}
      {(row as unknown as Record<string, unknown>)[col.key] ?? '—'}
    {/if}
  {/snippet}
</DataTable>

<Pager
  page={pageNum}
  perPage={PER_PAGE}
  {total}
  onpage={(p) => {
    pageNum = p;
    load();
  }}
/>

<ProductForm
  bind:open={formOpen}
  product={target}
  {categories}
  {brands}
  {units}
  products={allProducts}
  save={saveProduct}
  onsaved={() => {
    load();
    loadLookups();
  }}
/>

<style>
  .head {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 1rem;
  }
  h1 {
    margin: 0;
    font-size: 1.4rem;
  }
  .sub {
    margin: 0.2rem 0 0;
    color: var(--text-muted, #667);
    font-size: 0.85rem;
  }
  .actions {
    display: flex;
    gap: 0.5rem;
    flex-wrap: wrap;
  }
  .filters {
    display: flex;
    gap: 0.75rem;
    align-items: center;
    margin-bottom: 1rem;
  }
  .filters .q {
    flex: 1;
    max-width: 320px;
  }
  select {
    padding: 0.45rem 0.6rem;
    border: 1px solid var(--border, #d8dce3);
    border-radius: 6px;
    font: inherit;
  }
  .chk {
    display: flex;
    gap: 0.4rem;
    align-items: center;
    font-size: 0.85rem;
    color: var(--text-muted, #667);
  }
</style>
