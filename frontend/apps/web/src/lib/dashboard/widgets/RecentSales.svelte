<script lang="ts">
  import { onMount } from 'svelte';
  import { resolve } from '$app/paths';
  import { ApiError } from '@mauzo/api-client';
  import type { Sale } from '@mauzo/types';
  import { Icon } from '@mauzo/ui';
  import { formatMoney } from '@mauzo/utils';
  import { api } from '#lib/api.ts';
  import WidgetCard from '../WidgetCard.svelte';
  import type { WidgetProps } from '../types';

  let { widget, branchId }: WidgetProps = $props();

  let sales = $state<Sale[]>([]);
  let loading = $state(true);
  let error = $state<string | null>(null);

  onMount(async () => {
    try {
      const res = await api.list<Sale>('/api/v1/t/sales', {
        per_page: 7,
        ...(branchId ? { branch_id: branchId } : {}),
      });
      sales = res.data;
    } catch (e) {
      error = e instanceof ApiError ? e.message : 'Could not load sales';
    } finally {
      loading = false;
    }
  });
</script>

<WidgetCard title={widget.title} {loading} {error}>
  {#if sales.length === 0}
    <div class="none">
      <Icon name="receipt" size={28} />
      <p>No sales recorded yet.</p>
    </div>
  {:else}
    <ul>
      {#each sales as s (s.id)}
        <li>
          <a class="no" href={resolve('/sales')}>#{s.receipt_no}</a>
          <span class="who">{s.cashier || s.customer || 'Walk-in'}</span>
          <span class="amt">{formatMoney(s.total)}</span>
        </li>
      {/each}
    </ul>
    <a class="all" href={resolve('/sales')}>View all sales →</a>
  {/if}
</WidgetCard>

<style>
  ul {
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
  }
  li {
    display: grid;
    grid-template-columns: auto minmax(0, 1fr) auto;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-2) 0;
    border-bottom: 1px solid var(--color-border);
    font-size: var(--text-sm);
  }
  li:last-child {
    border-bottom: 0;
  }
  .no {
    font-weight: 600;
    color: var(--color-primary);
    text-decoration: none;
  }
  .no:hover {
    text-decoration: underline;
  }
  .who {
    color: var(--color-text-muted);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .amt {
    font-variant-numeric: tabular-nums;
    font-weight: 600;
  }
  .all {
    display: inline-block;
    margin-top: var(--space-3);
    font-size: var(--text-sm);
    color: var(--color-primary);
    text-decoration: none;
  }
  .all:hover {
    text-decoration: underline;
  }
  .none {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--space-2);
    min-height: 160px;
    color: var(--color-text-muted);
  }
  .none p {
    margin: 0;
    font-size: var(--text-sm);
  }
</style>
