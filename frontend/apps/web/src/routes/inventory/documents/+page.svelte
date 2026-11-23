<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import type { Branch } from '@mauzo/types';
  import { api } from '#lib/api.ts';
  import { auth } from '#lib/auth.svelte.ts';
  import AdjustmentsPanel from '#lib/inventory/AdjustmentsPanel.svelte';
  import TransfersPanel from '#lib/inventory/TransfersPanel.svelte';
  import CountsPanel from '#lib/inventory/CountsPanel.svelte';

  type Tab = 'adjustments' | 'transfers' | 'counts';
  const tabLabels: Record<Tab, string> = {
    adjustments: 'Adjustments',
    transfers: 'Transfers',
    counts: 'Stocktakes',
  };
  const TABS: Tab[] = ['adjustments', 'transfers', 'counts'];
  // The tab lives in the URL so the sidebar's "Stock transfers" link lands on it.
  const tab = $derived.by<Tab>(() => {
    const t = page.url.searchParams.get('tab');
    return TABS.includes(t as Tab) ? (t as Tab) : 'adjustments';
  });
  const switchTab = (t: Tab) =>
    // eslint-disable-next-line svelte/no-navigation-without-resolve -- query-only change on this page
    goto(t === 'adjustments' ? '?' : `?tab=${t}`, { replace: true, reset: false });
  let branches = $state<Branch[]>([]);
  let reload = $state(0);

  onMount(async () => {
    try {
      branches = await api.get<Branch[]>('/api/v1/t/branches/accessible');
    } catch {
      /* picker optional */
    }
  });

  const canSee = $derived(auth.can('inventory.stock.view'));
</script>

<div class="head">
  <div>
    <h1>Stock documents</h1>
    <p class="muted">Adjustments, inter-branch transfers and stocktakes.</p>
  </div>
</div>

{#if !canSee}
  <p class="muted">You don't have permission to view stock documents.</p>
{:else}
  <nav class="tabs">
    {#each ['adjustments', 'transfers', 'counts'] as t (t)}
      <button class:active={tab === t} onclick={() => switchTab(t as Tab)}>
        {tabLabels[t as Tab] ?? t}
      </button>
    {/each}
  </nav>

  {#if tab === 'adjustments'}
    <AdjustmentsPanel {branches} {reload} />
  {:else if tab === 'transfers'}
    <TransfersPanel {branches} {reload} />
  {:else}
    <CountsPanel {branches} {reload} />
  {/if}
{/if}

<style>
  .head {
    margin-bottom: 14px;
  }
  .muted {
    color: var(--fg-muted, #666);
    margin: 4px 0 0;
  }
  h1 {
    margin: 0;
  }
  .tabs {
    display: flex;
    gap: 4px;
    margin-bottom: 14px;
  }
  .tabs button {
    border: none;
    background: none;
    padding: 6px 12px;
    border-radius: 6px;
    cursor: pointer;
    color: var(--fg-muted, #666);
    font: inherit;
  }
  .tabs button.active {
    background: var(--surface-2, #f0f0ef);
    color: var(--fg, #222);
    font-weight: 600;
  }
</style>
