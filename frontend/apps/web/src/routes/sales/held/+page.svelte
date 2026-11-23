<script lang="ts">
  import { onMount } from 'svelte';
  import { ApiError } from '@mauzo/api-client';
  import type { HeldSale } from '@mauzo/types';
  import { DataTable, toast } from '@mauzo/ui';
  import { formatDate } from '@mauzo/utils';
  import { api } from '#lib/api.ts';

  let rows = $state<HeldSale[]>([]);
  let loading = $state(true);

  const columns = [
    { key: 'label', label: 'Label' },
    { key: 'held_by', label: 'Held by' },
    { key: 'branch', label: 'Branch' },
    { key: 'terminal', label: 'Terminal' },
    { key: 'lines', label: 'Lines', width: '80px' },
    { key: 'when', label: 'Held at', width: '150px' },
  ];

  function lineCount(cart: string): number {
    try {
      return (JSON.parse(cart) as unknown[]).length;
    } catch {
      return 0;
    }
  }

  async function load() {
    loading = true;
    try {
      rows = await api.get<HeldSale[]>('/api/v1/t/sales/held');
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Failed to load held sales');
    } finally {
      loading = false;
    }
  }

  onMount(load);
</script>

<div class="head">
  <h1>Held sales</h1>
  <p class="sub">Carts parked on the tills right now — recall or cancel them from the POS.</p>
</div>

<DataTable {columns} {rows} rowKey={(h) => h.id} {loading} empty="No held sales at the moment.">
  {#snippet cell(h: HeldSale, col)}
    {#if col.key === 'label'}<strong>{h.label || 'Held sale'}</strong>
    {:else if col.key === 'held_by'}{h.held_by || '—'}
    {:else if col.key === 'branch'}{h.branch}
    {:else if col.key === 'terminal'}{h.terminal}
    {:else if col.key === 'lines'}{lineCount(h.cart)}
    {:else if col.key === 'when'}{formatDate(h.held_at)}
    {/if}
  {/snippet}
</DataTable>

<style>
  h1 {
    font-size: 22px;
    font-weight: 700;
    margin: 0 0 4px;
  }
  .sub {
    color: var(--ink-3, #8a93a6);
    font-size: 13px;
    margin: 0 0 16px;
  }
</style>
