<script lang="ts">
  import { onMount, untrack } from 'svelte';
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import { ApiError } from '@mauzo/api-client';
  import type {
    Branch,
    ExpiringRow,
    ReorderRow,
    StockBalance,
    StockMovement,
    ValuationRow,
  } from '@mauzo/types';
  import { Button, Pager, toast, PageHeader } from '@mauzo/ui';
  import { api } from '#lib/api.ts';
  import { auth } from '#lib/auth.svelte.ts';
  import { branch } from '#lib/branch.svelte.ts';
  import StockOpModal from '#lib/inventory/StockOpModal.svelte';
  import ReportTabs from '#lib/inventory/ReportTabs.svelte';
  import LedgerTabs from '#lib/inventory/LedgerTabs.svelte';

  const PER_PAGE = 25;
  type Tab = 'stock' | 'movements' | 'reorders' | 'valuation';

  const tabLabels: Record<Tab, string> = {
    stock: 'Stock',
    movements: 'Movements',
    reorders: 'Reorder levels',
    valuation: 'Valuation',
  };
  let branches = $state<Branch[]>([]);
  let branchId = $state('');
  const TABS: Tab[] = ['stock', 'movements', 'reorders', 'valuation'];
  const urlTab = (): Tab => {
    const t = page.url.searchParams.get('tab');
    return TABS.includes(t as Tab) ? (t as Tab) : 'stock';
  };
  let tab = $state<Tab>(urlTab());
  let mounted = false;

  let balances = $state<StockBalance[]>([]);
  let movements = $state<StockMovement[]>([]);
  let reorders = $state<ReorderRow[]>([]);
  let valuations = $state<ValuationRow[]>([]);
  let expiring = $state<ExpiringRow[]>([]);
  let lowStock = $state(false);
  let pageNum = $state(1);
  let total = $state(0);
  let loading = $state(true);

  let receiveOpen = $state(false);
  let wasteOpen = $state(false);

  const canEdit = $derived(auth.can('inventory.adjustment.create'));
  const canWaste = $derived(auth.can('inventory.wastage.create'));

  async function load() {
    loading = true;
    try {
      const q = { page: pageNum, per_page: PER_PAGE, branch_id: branchId };
      if (tab === 'stock') {
        const res = await api.list<StockBalance>('/api/v1/t/inventory/balances', {
          ...q,
          low_stock: lowStock ? '1' : '',
        });
        balances = res.data;
        total = res.meta.total;
      } else if (tab === 'movements') {
        const res = await api.list<StockMovement>('/api/v1/t/inventory/movements', q);
        movements = res.data;
        total = res.meta.total;
      } else if (tab === 'reorders' && branchId) {
        reorders = await api.get<ReorderRow[]>(`/api/v1/t/inventory/reorders?branch_id=${branchId}`);
      } else if (tab === 'valuation') {
        valuations = await api.get<ValuationRow[]>('/api/v1/t/inventory/valuation');
      }
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Failed to load');
    } finally {
      loading = false;
    }
  }

  async function loadAlerts() {
    try {
      expiring = await api.get<ExpiringRow[]>('/api/v1/t/inventory/expiring?days=30');
    } catch {
      expiring = [];
    }
  }

  // Tabs live in the URL (?tab=) so sidebar links like "Stock movements"
  // land on the right one; the effect reacts to both kinds of navigation.
  function switchTab(t: Tab) {
    // eslint-disable-next-line svelte/no-navigation-without-resolve -- query-only change on this page
    goto(t === 'stock' ? '?' : `?tab=${t}`, { replace: true, reset: false });
  }

  $effect(() => {
    const next = urlTab();
    untrack(() => {
      if (!mounted || next === tab) return;
      tab = next;
      pageNum = 1;
      load();
    });
  });

  onMount(async () => {
    try {
      branches = await api.get<Branch[]>('/api/v1/t/branches/accessible');
      // Start on the branch picked in the top bar ("" = all branches).
      branchId = branch.id ?? '';
    } catch {
      /* branch picker optional */
    }
    mounted = true;
    await Promise.all([load(), loadAlerts()]);
  });
</script>

<PageHeader title="Stock" sub="On-hand balances, ledger movements and alerts." icon="layers">
  {#snippet actions()}
    {#if canEdit}<Button onclick={() => (receiveOpen = true)}>Receive stock</Button>{/if}
    {#if canWaste}<Button variant="ghost" onclick={() => (wasteOpen = true)}>Record wastage</Button>{/if}
  {/snippet}
</PageHeader>

{#if expiring.length > 0}
  <div class="alert">
    <strong>{expiring.length} batch{expiring.length === 1 ? '' : 'es'} expiring soon:</strong>
    {#each expiring.slice(0, 4) as x (x.batch_no)}
      <span class="pill">{x.product_name} · {x.batch_no} · {x.qty} u · {x.expiry_date?.slice(0, 10)}</span>
    {/each}
  </div>
{/if}

<div class="filters">
  <select
    bind:value={branchId}
    onchange={() => {
      pageNum = 1;
      load();
    }}
  >
    <option value="">All branches</option>
    {#each branches as b (b.id)}<option value={b.id}>{b.name}</option>{/each}
  </select>
  <nav class="tabs">
    {#each ['stock', 'movements', 'reorders', 'valuation'] as t (t)}
      <button class:active={tab === t} onclick={() => switchTab(t as Tab)}>
        {tabLabels[t as Tab] ?? t}
      </button>
    {/each}
  </nav>
  {#if tab === 'stock'}
    <label class="chk">
      <input type="checkbox" bind:checked={lowStock} onchange={load} /> Low stock only
    </label>
  {/if}
</div>

{#if tab === 'stock' || tab === 'movements'}
  <LedgerTabs {tab} {balances} {movements} {loading} />
{:else}
  <ReportTabs {tab} {branchId} {reorders} {valuations} {loading} />
{/if}

{#if tab === 'stock' || tab === 'movements'}
  <Pager
    page={pageNum}
    perPage={PER_PAGE}
    {total}
    onpage={(p) => {
      pageNum = p;
      load();
    }}
  />
{/if}

<StockOpModal
  kind="receive"
  {branches}
  bind:open={receiveOpen}
  ondone={() => {
    load();
    loadAlerts();
  }}
/>
<StockOpModal
  kind="wastage"
  {branches}
  bind:open={wasteOpen}
  ondone={() => {
    load();
    loadAlerts();
  }}
/>

<style>
  .alert {
    padding: 10px 14px;
    border: 1px solid var(--warn, #d97706);
    border-radius: 8px;
    margin-bottom: 12px;
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    align-items: center;
    font-size: 13px;
  }
  .pill {
    background: var(--surface-2, #c51b1b);
    border-radius: 99px;
    padding: 2px 10px;
    color: #f5f5f5;
    font-weight: 900;
  }

  .filters {
    display: flex;
    gap: 14px;
    align-items: center;
    margin-bottom: 12px;
    flex-wrap: wrap;
  }
  .filters select {
    padding: 6px 8px;
    border: 1px solid var(--border, #ddd);
    border-radius: 6px;
    background: var(--bg, #fff);
    color: var(--fg, #222);
  }
  .tabs {
    display: flex;
    gap: 4px;
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
  .chk {
    font-size: 13px;
    display: flex;
    gap: 6px;
    align-items: center;
    color: var(--fg-muted, #666);
  }
</style>
