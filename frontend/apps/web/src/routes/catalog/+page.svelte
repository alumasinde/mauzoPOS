<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { resolve } from '$app/paths';
  import { ApiError } from '@mauzo/api-client';
  import type { Brand, CatalogOverview, Category, Product, ProductDetail, Unit } from '@mauzo/types';
  import { Button, Icon, Pager, PageHeader, Select, TextField, toast } from '@mauzo/ui';
  import { api } from '#lib/api.ts';
  import { auth } from '#lib/auth.svelte.ts';
  import ProductForm from '#lib/catalog/ProductForm.svelte';
  import ProductsTable from '#lib/catalog/ProductsTable.svelte';
  import CatalogStats from '#lib/catalog/CatalogStats.svelte';
  import { importProducts } from '#lib/catalog/transfer.ts';
  import TransferMenu from '#lib/catalog/TransferMenu.svelte';

  const PER_PAGE = 25;
  let rows = $state<Product[]>([]);
  let categories = $state<Category[]>([]);
  let brands = $state<Brand[]>([]);
  let units = $state<Unit[]>([]);
  let allProducts = $state<{ id: string; sku: string; name: string }[]>([]);
  let stats = $state<CatalogOverview | null>(null);
  let loading = $state(true);
  let q = $state('');
  let categoryId = $state('');
  let brandId = $state('');
  let kind = $state('');
  let status = $state<'active' | 'archived' | 'all'>('active');
  let pageNum = $state(1);
  let total = $state(0);

  let formOpen = $state(false);
  let target = $state<ProductDetail | null>(null);
  let busy = $state<string | null>(null);
  let importInput = $state<HTMLInputElement>();

  const canManage = $derived(auth.can('catalog.product.create'));
  const canDelete = $derived(auth.can('catalog.product.delete'));

  const KINDS = [
    { v: 'standard', l: 'Standard' },
    { v: 'service', l: 'Service' },
    { v: 'packaged', l: 'Packaged' },
    { v: 'composite', l: 'Composite' },
  ] as const;

  async function load() {
    loading = true;
    try {
      const res = await api.list<Product>('/api/v1/t/catalog/products', {
        page: pageNum,
        per_page: PER_PAGE,
        q,
        category_id: categoryId,
        brand_id: brandId,
        kind,
        status,
      });
      rows = res.data;
      total = res.meta.total;
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Failed to load products');
    } finally {
      loading = false;
    }
  }

  async function loadStats() {
    try {
      stats = await api.get<CatalogOverview>('/api/v1/t/catalog/overview');
    } catch {
      stats = null; // cards are supplementary; the table is the real work
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

  function refilter() {
    pageNum = 1;
    load();
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
      loadStats();
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
      loadStats();
      loadLookups();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Import failed');
    } finally {
      if (importInput) importInput.value = '';
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
    loadStats();
    loadLookups();
  });
</script>

<svelte:head><title>Products — Mauzo</title></svelte:head>

<PageHeader title="Products" sub="Manage your catalogue, pricing and product availability" icon="box">
  {#snippet actions()}
    {#if canManage}
      <Button onclick={() => edit(null)}><Icon name="plus" size={16} /> Add product</Button>
      <TransferMenu onimport={() => importInput?.click()} onlabels={printLabels} />
      <input bind:this={importInput} type="file" accept=".csv,.xlsx" hidden onchange={doImport} />
    {/if}
  {/snippet}
</PageHeader>

{#if stats}<CatalogStats {stats} />{/if}

<div class="filters">
  <div class="q">
    <TextField placeholder="Search by name, SKU or barcode…" bind:value={q} oninput={refilter} />
  </div>
  <Select aria-label="Category" bind:value={categoryId} onchange={refilter}>
    <option value="">All Categories</option>
    {#each categories as c (c.id)}<option value={c.id}>{c.name}</option>{/each}
  </Select>
  <Select aria-label="Brand" bind:value={brandId} onchange={refilter}>
    <option value="">All Brands</option>
    {#each brands as b (b.id)}<option value={b.id}>{b.name}</option>{/each}
  </Select>
  <Select aria-label="Type" bind:value={kind} onchange={refilter}>
    <option value="">All Types</option>
    {#each KINDS as k (k.v)}<option value={k.v}>{k.l}</option>{/each}
  </Select>
  <Select aria-label="Status" bind:value={status} onchange={refilter}>
    <option value="active">Active</option>
    <option value="archived">Archived</option>
    <option value="all">All Statuses</option>
  </Select>
</div>

<ProductsTable {rows} {loading} {canManage} {canDelete} {busy} onedit={edit} onarchive={archive} />

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
    loadStats();
    loadLookups();
  }}
/>

<style>
  .filters {
    display: flex;
    gap: var(--space-2);
    align-items: center;
    margin-bottom: var(--space-4);
    flex-wrap: wrap;
  }
  .filters .q {
    flex: 1;
    min-width: 220px;
    max-width: 340px;
  }
</style>
