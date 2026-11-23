<script lang="ts">
  import { PageHeader } from '@mauzo/ui';
  import { onDestroy, onMount } from 'svelte';
  import AssignmentsPanel from '#lib/receipts/AssignmentsPanel.svelte';
  import { Designer } from '#lib/receipts/designer.svelte.ts';
  import LayoutsTab from '#lib/receipts/LayoutsTab.svelte';
  import LogoPanel from '#lib/receipts/LogoPanel.svelte';
  import ReceiptSettings from '#lib/receipts/ReceiptSettings.svelte';

  type Tab = 'layouts' | 'branches' | 'details';
  let tab = $state<Tab>('layouts');
  const d = new Designer();

  onMount(() => d.load());
  onDestroy(() => d.dispose());
</script>

<PageHeader
  title="Receipt designer"
  sub="Design what customers get from the till. Changes reach each till on its next sync."
  icon="printer"
/>

<nav class="tabs" aria-label="Receipt designer sections">
  <button class:active={tab === 'layouts'} onclick={() => (tab = 'layouts')}>Layouts</button>
  <button class:active={tab === 'branches'} onclick={() => (tab = 'branches')}>Branches</button>
  <button class:active={tab === 'details'} onclick={() => (tab = 'details')}>Logo &amp; details</button>
</nav>

{#if d.loading}
  <p class="muted">Loading…</p>
{:else if tab === 'layouts'}
  <LayoutsTab {d} />
{:else if tab === 'branches'}
  <div class="card"><AssignmentsPanel templates={d.templates} /></div>
{:else}
  <div class="card">
    <h2>Logo</h2>
    <LogoPanel logoUrl={d.logoUrl} onchanged={() => d.refreshLogo()} />
  </div>
  <div class="card"><ReceiptSettings onsaved={() => d.schedulePreview(0)} /></div>
{/if}

<style>
  .muted {
    color: var(--color-text-muted);
    margin: 0;
  }
  .tabs {
    display: flex;
    gap: var(--space-1);
    margin: var(--space-4) 0;
    border-bottom: 1px solid var(--color-border);
  }
  .tabs button {
    padding: var(--space-2) var(--space-4);
    background: none;
    border: 0;
    border-bottom: 2px solid transparent;
    font: inherit;
    color: var(--color-text-muted);
    cursor: pointer;
  }
  .tabs button.active {
    color: var(--color-primary);
    border-bottom-color: var(--color-primary);
    font-weight: 600;
  }
  .card {
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-lg);
    padding: var(--space-4) var(--space-5);
    box-shadow: var(--shadow-sm);
    margin-bottom: var(--space-4);
  }
  h2 {
    margin: 0 0 var(--space-3);
    font-size: var(--text-lg);
  }
</style>
