<script lang="ts">
  import type { ReorderRow, ValuationRow } from '@mauzo/types';
  import { Badge, DataTable } from '@mauzo/ui';

  interface Props {
    tab: 'reorders' | 'valuation';
    branchId: string;
    reorders: ReorderRow[];
    valuations: ValuationRow[];
    loading: boolean;
  }
  let { tab, branchId, reorders, valuations, loading }: Props = $props();

  const roCols = [
    { key: 'product_name', label: 'Product' },
    { key: 'sku', label: 'SKU', width: '110px' },
    { key: 'min_qty', label: 'Min', width: '80px', align: 'right' as const },
    { key: 'target_qty', label: 'Target', width: '80px', align: 'right' as const },
    { key: 'qty', label: 'On hand', width: '90px', align: 'right' as const },
  ];
  const valCols = [
    { key: 'branch_name', label: 'Branch' },
    { key: 'sku_count', label: 'SKUs', width: '90px', align: 'right' as const },
    { key: 'units', label: 'Units', width: '100px', align: 'right' as const },
    { key: 'value', label: 'Value (KES)', width: '130px', align: 'right' as const },
  ];
</script>

{#if tab === 'reorders'}
  {#if !branchId}
    <p class="muted">Pick a branch to see its reorder levels.</p>
  {:else}
    <DataTable
      columns={roCols}
      rows={reorders}
      rowKey={(r) => `${r.product_id}:${r.variant_id ?? 0}`}
      {loading}
      empty="No reorder levels set."
    >
      {#snippet cell(row, col)}
        {#if col.key === 'qty'}
          <Badge tone={Number(row.qty) <= Number(row.min_qty) ? 'warning' : 'success'}>{row.qty}</Badge>
        {:else}
          {(row as unknown as Record<string, unknown>)[col.key] ?? '—'}
        {/if}
      {/snippet}
    </DataTable>
  {/if}
{:else}
  <DataTable
    columns={valCols}
    rows={valuations}
    rowKey={(r) => r.branch_id}
    {loading}
    empty="No stock to value."
  >
    {#snippet cell(row, col)}
      {#if col.key === 'value'}
        <strong>KES {row.value}</strong>
      {:else}
        {(row as unknown as Record<string, unknown>)[col.key] ?? '—'}
      {/if}
    {/snippet}
  </DataTable>
{/if}

<style>
  .muted {
    color: var(--fg-muted, #666);
  }
</style>
