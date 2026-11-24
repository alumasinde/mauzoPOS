<script lang="ts">
  // M-Pesa reconciliation — the feed of collections, their allocation state,
  // and the tools to link, dispute or reverse them.
  import { onMount } from 'svelte';
  import { ApiError } from '@mauzo/api-client';
  import { Badge, Button, DataTable, Icon, PageHeader, Pager, Select, StatCard, toast } from '@mauzo/ui';
  import { formatMoney } from '@mauzo/utils';
  import { auth } from '#lib/auth.svelte.ts';
  import {
    mpesaApi,
    type MpesaSuggestion,
    type MpesaSummary,
    type MpesaTxn,
    type TxnFilters,
  } from '#lib/mpesa/api.ts';
  import TxnDetail from '#lib/mpesa/TxnDetail.svelte';
  import SettingsModal from '#lib/mpesa/SettingsModal.svelte';

  const TABS = [
    { key: 'awaiting', label: 'Awaiting allocation' },
    { key: 'allocated', label: 'Allocated' },
    { key: 'issues', label: 'Failed / cancelled' },
    { key: 'reversals', label: 'Reversals' },
    { key: 'all', label: 'All' },
  ] as const;

  let tab = $state<(typeof TABS)[number]['key']>('awaiting');
  let summary = $state<MpesaSummary | null>(null);
  let rows = $state<MpesaTxn[]>([]);
  let total = $state(0);
  let page = $state(1);
  let loading = $state(true);
  let q = $state('');
  let kind = $state('');
  let from = $state('');
  let to = $state('');
  let detail = $state<MpesaTxn | null>(null);
  let suggestions = $state<MpesaSuggestion[]>([]);
  let settingsOpen = $state(false);

  const canConfig = $derived(auth.can('mpesa.config.manage'));
  const canAllocate = $derived(auth.can('mpesa.transaction.allocate'));
  const canReallocate = $derived(auth.can('mpesa.transaction.reallocate'));

  const columns = [
    { key: 'when', label: 'Received', width: '150px' },
    { key: 'code', label: 'Code', width: '120px' },
    { key: 'payer', label: 'Payer' },
    { key: 'amount', label: 'Amount', width: '110px', align: 'right' as const },
    { key: 'kind', label: 'Type', width: '90px' },
    { key: 'status', label: 'Status', width: '130px' },
    { key: 'allocated', label: 'Sale', width: '100px' },
  ];

  function filters(): TxnFilters {
    const f: TxnFilters = { q, kind, from, to };
    if (tab === 'awaiting') f.allocated = 'no';
    else if (tab === 'allocated') f.allocated = 'yes';
    else if (tab === 'issues') f.status = 'failed';
    else if (tab === 'reversals') f.status = 'reversal_pending,reversed';
    return f;
  }

  async function load() {
    loading = true;
    try {
      const f = filters();
      if (f.status?.includes(',')) {
        // The API takes one status — fetch both sets for the reversals tab.
        const [pend, rev] = await Promise.all([
          mpesaApi.list({ ...f, status: 'reversal_pending' }, page),
          mpesaApi.list({ ...f, status: 'reversed' }, page),
        ]);
        rows = [...pend.items, ...rev.items].sort((a, b) => b.created_at.localeCompare(a.created_at));
        total = pend.total + rev.total;
      } else {
        const res = await mpesaApi.list(f, page);
        rows = res.items;
        total = res.total;
      }
      const [s, sug] = await Promise.all([
        mpesaApi.summary(),
        mpesaApi.suggestions().catch(() => ({ items: [] })),
      ]);
      summary = s;
      suggestions = sug.items;
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Could not load M-Pesa transactions');
    } finally {
      loading = false;
    }
  }

  function switchTab(t: typeof tab) {
    tab = t;
    page = 1;
    void load();
  }

  function when(t: MpesaTxn): string {
    const d = t.trans_time ?? t.created_at;
    return d.replace('T', ' ').slice(5, 16);
  }

  const toneFor: Record<string, 'neutral' | 'info' | 'success' | 'warning' | 'danger'> = {
    success: 'success',
    failed: 'danger',
    cancelled: 'warning',
    timeout: 'warning',
    pending: 'info',
    initiated: 'info',
    reversal_pending: 'warning',
    reversed: 'danger',
  };

  const strongSugg = $derived(suggestions.filter((s) => s.strength === 'strong'));

  onMount(load);
</script>

<PageHeader title="M-Pesa reconciliation" icon="phone" sub="Collections, allocations and reversals">
  {#snippet actions()}
    <Button variant="secondary" onclick={load}><Icon name="refresh" size={16} /> Refresh</Button>
    {#if canConfig}
      <Button variant="primary" onclick={() => (settingsOpen = true)}>
        <Icon name="settings" size={16} /> M-Pesa settings
      </Button>
    {/if}
  {/snippet}
</PageHeader>

<div class="stats">
  <StatCard
    label="Collected today"
    value="KSh {formatMoney(summary?.collected_today ?? '0')}"
    icon="banknote"
    tone="green"
  />
  <StatCard label="Payments today" value={String(summary?.count_today ?? 0)} icon="receipt" tone="blue" />
  <StatCard
    label="Awaiting allocation"
    value={String(summary?.unallocated ?? 0)}
    icon="inbox"
    tone="orange"
  />
  <StatCard label="Failed (7 days)" value={String(summary?.failed_7d ?? 0)} icon="alert" tone="red" />
</div>

{#if strongSugg.length > 0}
  <div class="sugg">
    <Icon name="info" size={16} />
    <span>
      <b>{strongSugg.length}</b> payment{strongSugg.length === 1 ? '' : 's'} look
      {strongSugg.length === 1 ? 's' : ''} like exact matches — codes already cited on receipts. Review them in
      the Awaiting allocation tab.
    </span>
  </div>
{/if}

<div class="tabs">
  {#each TABS as t (t.key)}
    <button class:active={tab === t.key} onclick={() => switchTab(t.key)}>{t.label}</button>
  {/each}
</div>

<div class="filters">
  <input
    class="q"
    bind:value={q}
    placeholder="Search code, phone, ref…"
    onkeydown={(e) => e.key === 'Enter' && load()}
  />
  <Select bind:value={kind} aria-label="Type">
    <option value="">All types</option>
    <option value="stk">STK push</option>
    <option value="c2b">Till / Paybill</option>
  </Select>
  <input type="date" bind:value={from} aria-label="From" />
  <input type="date" bind:value={to} aria-label="To" />
  <Button
    variant="secondary"
    onclick={() => {
      page = 1;
      void load();
    }}
  >
    Apply
  </Button>
</div>

<DataTable
  {columns}
  {rows}
  rowKey={(r) => r.public_id}
  {loading}
  empty="No transactions match."
  onrowclick={(r) => (detail = r)}
>
  {#snippet cell(row: MpesaTxn, col: { key: string })}
    {#if col.key === 'when'}
      <span class="when">{when(row)}</span>
    {:else if col.key === 'code'}
      {#if row.trans_id}<code class="code">{row.trans_id}</code>{:else}<span class="muted">—</span>{/if}
    {:else if col.key === 'payer'}
      <div class="payer">
        <b>{row.payer_name || row.phone || '—'}</b>
        {#if row.account_ref}<small>ref {row.account_ref}</small>{/if}
      </div>
    {:else if col.key === 'amount'}
      <b>{formatMoney(row.amount)}</b>
    {:else if col.key === 'kind'}
      <span class="muted">{row.kind === 'stk' ? 'STK push' : 'C2B'}</span>
    {:else if col.key === 'status'}
      <Badge tone={toneFor[row.status] ?? 'neutral'}>{row.status.replace('_', ' ')}</Badge>
    {:else if col.key === 'allocated'}
      {#if row.allocated_sale_id}
        <Badge tone="success"><Icon name="check" size={12} /> linked</Badge>
      {:else if row.status === 'success'}
        <Badge tone="warning">unallocated</Badge>
      {:else}<span class="muted">—</span>{/if}
    {/if}
  {/snippet}
</DataTable>

<div class="foot">
  <Pager
    {page}
    {total}
    perPage={25}
    onpage={(p) => {
      page = p;
      void load();
    }}
  />
</div>

{#if detail}
  <TxnDetail
    bind:txn={detail}
    {suggestions}
    {canAllocate}
    {canReallocate}
    onchange={() => {
      detail = null;
      void load();
    }}
  />
{/if}
<SettingsModal bind:open={settingsOpen} onsaved={() => void load()} />

<style>
  .stats {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: var(--space-3);
    margin-bottom: var(--space-4);
  }
  .sugg {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    background: color-mix(in srgb, var(--color-primary) 7%, transparent);
    border: 1px solid color-mix(in srgb, var(--color-primary) 25%, transparent);
    border-radius: var(--radius-md);
    padding: var(--space-3);
    margin-bottom: var(--space-3);
    font-size: var(--text-sm);
  }
  .tabs {
    display: flex;
    gap: var(--space-1);
    border-bottom: 1px solid var(--color-border);
    margin-bottom: var(--space-3);
  }
  .tabs button {
    border: 0;
    background: none;
    padding: var(--space-2) var(--space-3);
    font: inherit;
    font-size: var(--text-sm);
    font-weight: 600;
    color: var(--color-text-muted);
    cursor: pointer;
    border-bottom: 2px solid transparent;
    margin-bottom: -1px;
  }
  .tabs button.active {
    color: var(--color-primary);
    border-bottom-color: var(--color-primary);
  }
  .filters {
    display: flex;
    gap: var(--space-2);
    align-items: center;
    margin-bottom: var(--space-3);
    flex-wrap: wrap;
  }
  .filters .q {
    flex: 1;
    min-width: 200px;
    padding: 8px 12px;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    font: inherit;
    background: var(--color-surface);
  }
  .filters input[type='date'] {
    padding: 7px 10px;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    font: inherit;
    background: var(--color-surface);
  }
  .when {
    font-variant-numeric: tabular-nums;
    font-size: var(--text-sm);
  }
  .code {
    font-weight: 700;
    color: var(--color-primary);
  }
  .payer {
    display: flex;
    flex-direction: column;
  }
  .payer small {
    color: var(--color-text-muted);
  }
  .muted {
    color: var(--color-text-muted);
  }
  .foot {
    display: flex;
    justify-content: flex-end;
    margin-top: var(--space-3);
  }
  /* make rows clickable */
  :global(.mpesa-row) {
    cursor: pointer;
  }
</style>
