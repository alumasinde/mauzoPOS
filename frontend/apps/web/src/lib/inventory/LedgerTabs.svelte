<script lang="ts">
  import type { StockBalance, StockMovement } from '@mauzo/types';
  import { Badge, DataTable } from '@mauzo/ui';

  interface Props {
    tab: 'stock' | 'movements';
    balances: StockBalance[];
    movements: StockMovement[];
    loading: boolean;
  }
  let { tab, balances, movements, loading }: Props = $props();

  const balCols = [
    { key: 'product_name', label: 'Product' },
    { key: 'location_name', label: 'Location', width: '140px' },
    { key: 'branch_name', label: 'Branch', width: '140px' },
    { key: 'qty', label: 'On hand', width: '100px', align: 'right' as const },
    { key: 'avg_cost', label: 'Avg cost', width: '100px', align: 'right' as const },
  ];
  const movCols = [
    { key: 'created_at', label: 'When', width: '150px' },
    { key: 'product', label: 'Product' },
    { key: 'qty', label: 'Qty', width: '80px', align: 'right' as const },
    { key: 'kind', label: 'Kind', width: '110px' },
    { key: 'reason', label: 'Reason', width: '100px' },
    { key: 'location_name', label: 'Location', width: '130px' },
  ];

  const fmt = (s?: string | null) =>
    s ? new Date(s).toLocaleString(undefined, { dateStyle: 'short', timeStyle: 'short' }) : '—';
</script>

{#if tab === 'stock'}
  <DataTable
    columns={balCols}
    rows={balances}
    rowKey={(r) => `${r.product_id}:${r.location_id}:${r.batch_id ?? 0}`}
    {loading}
    empty="No stock yet — receive some to get started."
  >
    {#snippet cell(row, col)}
      {#if col.key === 'qty'}
        <strong>{row.qty}</strong>{row.batch_id ? ' (batch)' : ''}
      {:else}
        {(row as unknown as Record<string, unknown>)[col.key] ?? '—'}
      {/if}
    {/snippet}
  </DataTable>
{:else}
  <DataTable columns={movCols} rows={movements} rowKey={(r) => r.id} {loading} empty="No movements yet.">
    {#snippet cell(row, col)}
      {#if col.key === 'created_at'}
        {fmt(row.created_at)}
      {:else if col.key === 'qty'}
        <span class:neg={row.qty.startsWith('-')} class="qty">{row.qty}</span>
      {:else if col.key === 'kind'}
        <Badge tone={row.kind === 'waste' ? 'warning' : 'neutral'}>{row.kind}</Badge>
      {:else}
        {(row as unknown as Record<string, unknown>)[col.key] ?? '—'}
      {/if}
    {/snippet}
  </DataTable>
{/if}

<style>
  .qty {
    font-variant-numeric: tabular-nums;
  }
  .neg {
    color: var(--danger, #dc2626);
  }
</style>
