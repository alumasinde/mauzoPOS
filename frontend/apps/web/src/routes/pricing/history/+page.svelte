<script lang="ts">
  import { onMount } from 'svelte';
  import { ApiError } from '@mauzo/api-client';
  import type { PriceHistoryRow, Product } from '@mauzo/types';
  import { DataTable, Pager, toast, PageHeader } from '@mauzo/ui';
  import { api } from '#lib/api.ts';

  const PER_PAGE = 25;
  const columns = [
    { key: 'created_at', label: 'When', width: '160px' },
    { key: 'product_name', label: 'Product' },
    { key: 'source', label: 'Source', width: '110px' },
    { key: 'old_price', label: 'Old', width: '110px', align: 'right' as const },
    { key: 'new_price', label: 'New', width: '110px', align: 'right' as const },
  ];

  let products = $state<Product[]>([]);
  let productId = $state('');
  let rows = $state<PriceHistoryRow[]>([]);
  let pageNum = $state(1);
  let total = $state(0);
  let loading = $state(false);

  async function load() {
    if (!productId) {
      rows = [];
      total = 0;
      return;
    }
    loading = true;
    try {
      const res = await api.list<PriceHistoryRow>('/api/v1/t/pricing/history', {
        product_id: productId,
        page: pageNum,
        per_page: PER_PAGE,
      });
      rows = res.data;
      total = res.meta.total;
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Failed to load history');
    } finally {
      loading = false;
    }
  }

  onMount(async () => {
    try {
      const res = await api.list<Product>('/api/v1/t/catalog/products', { per_page: 200 });
      products = res.data;
    } catch {
      /* optional */
    }
  });

  const fmt = (s?: string | null) =>
    s ? new Date(s).toLocaleString(undefined, { dateStyle: 'short', timeStyle: 'short' }) : '—';
</script>

<PageHeader title="Price history" sub="Every recorded price change, per product." icon="history" />

<div class="filters">
  <select
    bind:value={productId}
    onchange={() => {
      pageNum = 1;
      load();
    }}
  >
    <option value="">Pick a product…</option>
    {#each products as p (p.id)}<option value={p.id}>{p.name} ({p.sku})</option>{/each}
  </select>
</div>

<DataTable
  {columns}
  {rows}
  rowKey={(r) => `${r.created_at}:${r.new_price}`}
  {loading}
  empty={productId ? 'No price changes recorded.' : 'Pick a product to see its history.'}
>
  {#snippet cell(row, col)}
    {#if col.key === 'created_at'}
      {fmt(row.created_at)}
    {:else if col.key === 'old_price'}
      {row.old_price ?? '—'}
    {:else if col.key === 'new_price'}
      <strong>{row.new_price}</strong>
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

<style>
  .filters {
    margin-bottom: 12px;
  }
  .filters select {
    padding: 6px 8px;
    border: 1px solid var(--border, #ddd);
    border-radius: 6px;
    background: var(--bg, #fff);
    color: var(--fg, #222);
    min-width: 280px;
  }
</style>
