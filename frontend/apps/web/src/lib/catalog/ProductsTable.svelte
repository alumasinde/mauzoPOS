<script lang="ts">
  // Products table — thumbnail, category chip, stock count and status badge.
  import type { Product } from '@mauzo/types';
  import { Badge, DataTable, Icon } from '@mauzo/ui';
  import { categoryIcon, formatMoney } from '@mauzo/utils';
  import ProductActions from './ProductActions.svelte';

  interface Props {
    rows: Product[];
    loading: boolean;
    canManage: boolean;
    canDelete: boolean;
    busy: string | null;
    onedit: (p: Product | null) => void;
    onarchive: (p: Product) => void;
  }
  let { rows, loading, canManage, canDelete, busy, onedit, onarchive }: Props = $props();

  const columns = [
    { key: 'sku', label: 'SKU', width: '110px' },
    { key: 'name', label: 'Product' },
    { key: 'category_name', label: 'Category', width: '140px' },
    { key: 'kind', label: 'Type', width: '100px' },
    { key: 'unit', label: 'Unit', width: '70px' },
    { key: 'price', label: 'Selling price', width: '110px', align: 'right' as const },
    { key: 'stock', label: 'Stock', width: '80px', align: 'right' as const },
    { key: 'status', label: 'Status', width: '100px' },
    { key: 'actions', label: '', width: '90px', align: 'right' as const },
  ];
</script>

<DataTable
  {columns}
  {rows}
  rowKey={(r) => r.id}
  {loading}
  empty="No products yet — import a CSV or create one."
  onrowclick={canManage ? onedit : undefined}
>
  {#snippet cell(row, col)}
    {#if col.key === 'name'}
      <span class="prod">
        {#if row.image_url}
          <img class="thumb" src={row.image_url} alt="" />
        {:else}
          <span class="thumb ph" aria-hidden="true">
            <Icon name={categoryIcon(row.category_name ?? row.name)} size={18} />
          </span>
        {/if}
        <span class="pname">{row.name}</span>
      </span>
    {:else if col.key === 'category_name'}
      {#if row.category_name}
        <span class="cat"><Icon name={categoryIcon(row.category_name)} size={15} />{row.category_name}</span>
      {:else}
        —
      {/if}
    {:else if col.key === 'kind'}
      <Badge tone={row.kind === 'composite' ? 'info' : 'neutral'}>{row.kind}</Badge>
    {:else if col.key === 'unit'}
      {row.sell_by === 'weight' ? `per ${row.unit_code ?? 'unit'}` : (row.unit_code ?? 'unit')}
    {:else if col.key === 'price'}
      KSh {formatMoney(row.price)}
    {:else if col.key === 'stock'}
      {#if row.stock !== null}
        <span class="stocknum" class:low={row.low_stock}>{Number(row.stock)}</span>
      {:else}
        —
      {/if}
    {:else if col.key === 'status'}
      {#if row.status === 'archived'}
        <Badge tone="neutral">Archived</Badge>
      {:else if row.low_stock}
        <Badge tone="warning">Low stock</Badge>
      {:else}
        <Badge tone="success">In stock</Badge>
      {/if}
    {:else if col.key === 'actions'}
      <ProductActions row={row as Product} {canManage} {canDelete} {busy} {onedit} {onarchive} />
    {:else}
      {(row as unknown as Record<string, unknown>)[col.key] ?? '—'}
    {/if}
  {/snippet}
</DataTable>

<style>
  .prod {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    min-width: 0;
  }
  .thumb {
    width: 32px;
    height: 32px;
    border-radius: var(--radius-sm);
    object-fit: cover;
    border: 1px solid var(--color-border);
    flex: none;
  }
  .thumb.ph {
    display: grid;
    place-items: center;
    background: var(--color-surface-muted);
    color: var(--color-text-muted);
  }
  .pname {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .cat {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    color: var(--color-text-muted);
  }
  .stocknum.low {
    color: var(--color-warning);
    font-weight: 700;
  }
</style>
