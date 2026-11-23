<script lang="ts">
  import { onMount } from 'svelte';
  import { ApiError } from '@mauzo/api-client';
  import type { Branch, Sale, SaleSummary } from '@mauzo/types';
  import { Badge, Button, DataTable, Pager, toast } from '@mauzo/ui';
  import { formatDate, formatMoney } from '@mauzo/utils';
  import { api } from '#lib/api.ts';
  import SaleDetail from '#lib/sales/SaleDetail.svelte';

  interface Props {
    title?: string;
    subtitle?: string;
    showKpis?: boolean;
    defaultFrom?: string;
    defaultTo?: string;
  }
  let { title = 'Transactions', subtitle = '', showKpis = true, defaultFrom, defaultTo }: Props = $props();

  const PER_PAGE = 25;
  let rows = $state<Sale[]>([]);
  let branches = $state<Branch[]>([]);
  let summary = $state<SaleSummary | null>(null);
  let loading = $state(true);
  let pageNum = $state(1);
  let total = $state(0);

  // filters
  let search = $state('');
  let status = $state('');
  let method = $state('');
  let branchId = $state('');
  let from = $state('');
  let to = $state('');

  let detailId = $state<string | null>(null);
  let detailOpen = $state(false);

  function today() {
    return new Date().toISOString().slice(0, 10);
  }

  function params() {
    const p: Record<string, string | number> = {
      page: pageNum,
      per_page: PER_PAGE,
      search,
      status,
      method,
    };
    if (branchId) p.branch_id = branchId;
    if (from) p.from = from;
    if (to) p.to = to;
    return p;
  }

  async function load() {
    loading = true;
    try {
      const [res, sum] = await Promise.all([
        api.list<Sale>('/api/v1/t/sales', params()),
        api.get<SaleSummary>('/api/v1/t/sales/summary?' + qs()).catch(() => null),
      ]);
      rows = res.data;
      total = res.meta.total;
      summary = sum;
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Failed to load sales');
    } finally {
      loading = false;
    }
  }

  function qs() {
    return Object.entries(params())
      .filter(([k]) => !['page', 'per_page'].includes(k))
      .map(([k, v]) => `${k}=${encodeURIComponent(v)}`)
      .join('&');
  }

  function reload() {
    pageNum = 1;
    load();
  }

  const columns = [
    { key: 'receipt', label: 'Receipt', width: '90px' },
    { key: 'when', label: 'Sold at', width: '150px' },
    { key: 'branch', label: 'Branch' },
    { key: 'cashier', label: 'Cashier' },
    { key: 'customer', label: 'Customer' },
    { key: 'status', label: 'Status', width: '120px' },
    { key: 'total', label: 'Total', width: '130px', align: 'right' as const },
  ];

  const TONE = {
    completed: 'success',
    voided: 'danger',
    returned: 'neutral',
    partially_returned: 'warning',
  } as const;

  onMount(async () => {
    from = defaultFrom ?? today();
    to = defaultTo ?? today();
    load();
    branches = await api.get<Branch[]>('/api/v1/t/branches').catch(() => []);
  });
</script>

<div class="head">
  <div>
    <h1>{title}</h1>
    <p class="sub">{subtitle}</p>
  </div>
  {#if summary && showKpis}
    <div class="kpis">
      <div class="kpi"><span>Total</span><strong>{formatMoney(summary.total)}</strong></div>
      <div class="kpi"><span>Count</span><strong>{summary.cnt}</strong></div>
      <div class="kpi"><span>Average</span><strong>{formatMoney(summary.avg)}</strong></div>
    </div>
  {/if}
</div>

<div class="filters">
  <input
    class="search"
    placeholder="Receipt no, customer or item…"
    bind:value={search}
    onkeydown={(e) => e.key === 'Enter' && reload()}
  />
  <select bind:value={status} onchange={reload}>
    <option value="">All statuses</option>
    <option value="completed">Completed</option>
    <option value="partially_returned">Partially returned</option>
    <option value="returned">Returned</option>
    <option value="voided">Voided</option>
  </select>
  <select bind:value={method} onchange={reload}>
    <option value="">All methods</option>
    <option value="cash">Cash</option>
    <option value="card">Card</option>
    <option value="bank">Bank transfer</option>
    <option value="credit">Credit</option>
  </select>
  <select bind:value={branchId} onchange={reload}>
    <option value="">All branches</option>
    {#each branches as b (b.id)}<option value={b.id}>{b.name}</option>{/each}
  </select>
  <input type="date" bind:value={from} onchange={reload} />
  <input type="date" bind:value={to} onchange={reload} />
  <Button variant="secondary" onclick={reload}>Apply</Button>
</div>

<DataTable
  {columns}
  {rows}
  rowKey={(r) => r.id}
  {loading}
  empty="No sales in this window."
  onrowclick={(s) => {
    detailId = s.id;
    detailOpen = true;
  }}
>
  {#snippet cell(s: Sale, col)}
    {#if col.key === 'receipt'}<strong>#{s.receipt_no}</strong>
    {:else if col.key === 'when'}{formatDate(s.sold_at)}
    {:else if col.key === 'branch'}{s.branch}
    {:else if col.key === 'cashier'}{s.cashier || '—'}
    {:else if col.key === 'customer'}{s.customer || 'Walk-in'}
    {:else if col.key === 'status'}<Badge tone={TONE[s.status]}>{s.status.replace('_', ' ')}</Badge>
    {:else if col.key === 'total'}<span class="num">{formatMoney(s.total)}</span>
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

<SaleDetail bind:open={detailOpen} saleId={detailId} onchanged={load} />

<style>
  .head {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 16px;
  }
  h1 {
    font-size: 22px;
    font-weight: 700;
    margin: 0;
  }
  .sub {
    color: var(--ink-3, #8a93a6);
    font-size: 13px;
    margin: 4px 0 0;
  }
  .kpis {
    display: flex;
    gap: 20px;
  }
  .kpi {
    text-align: right;
  }
  .kpi span {
    display: block;
    font-size: 11px;
    color: var(--ink-3, #8a93a6);
    text-transform: uppercase;
  }
  .kpi strong {
    font-size: 17px;
  }
  .filters {
    display: flex;
    gap: 8px;
    margin-bottom: 14px;
    flex-wrap: wrap;
  }
  .filters input,
  .filters select {
    padding: 8px 10px;
    border: 1px solid var(--line, #e3e7ee);
    border-radius: 8px;
    font: inherit;
    font-size: 13px;
  }
  .search {
    flex: 1;
    min-width: 220px;
  }
  .num {
    text-align: right;
    font-variant-numeric: tabular-nums;
  }
</style>
