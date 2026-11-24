<script lang="ts">
  // Cashier shifts — open drawers across tills, close reports, cash variance
  // review. The back-office mirror of what happens at the till.
  import { onMount } from 'svelte';
  import { ApiError } from '@mauzo/api-client';
  import { Badge, DataTable, PageHeader, Pager, Select, StatCard, toast } from '@mauzo/ui';
  import { formatDate, formatMoney } from '@mauzo/utils';
  import type { Branch } from '@mauzo/types';
  import { api } from '#lib/api.ts';
  import { branch } from '#lib/branch.svelte.ts';
  import { shiftsApi, type Shift, type ShiftSummary } from '#lib/shifts/api.ts';
  import ShiftDetail from '#lib/shifts/ShiftDetail.svelte';

  let rows = $state<Shift[]>([]);
  let summary = $state<ShiftSummary | null>(null);
  let branches = $state<Branch[]>([]);
  let total = $state(0);
  let page = $state(1);
  let loading = $state(true);
  let status = $state('');
  let branchID = $state('');
  let detail = $state<Shift | null>(null);

  const columns = [
    { key: 'opened', label: 'Opened', width: '150px' },
    { key: 'till', label: 'Till' },
    { key: 'cashier', label: 'Cashier' },
    { key: 'branch', label: 'Branch' },
    { key: 'float', label: 'Float', width: '110px', align: 'right' as const },
    { key: 'expected', label: 'Expected', width: '120px', align: 'right' as const },
    { key: 'variance', label: 'Variance', width: '130px' },
    { key: 'status', label: 'Status', width: '110px' },
  ];

  async function load() {
    loading = true;
    try {
      const [list, sum] = await Promise.all([
        shiftsApi.list({ status, branch_id: branchID }, page),
        shiftsApi.summary(),
      ]);
      rows = list.items ?? [];
      total = list.total;
      summary = sum;
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Failed to load shifts');
    } finally {
      loading = false;
    }
  }

  function varianceTone(s: Shift): 'success' | 'danger' | 'info' | 'neutral' {
    if (s.variance === null) return 'neutral';
    const v = Number(s.variance);
    if (v === 0) return 'success';
    return v > 0 ? 'info' : 'danger';
  }

  onMount(async () => {
    branches = await api.get<Branch[]>('/api/v1/t/branches/accessible').catch(() => [] as Branch[]);
  });

  $effect(() => {
    void branch.id; // reload when the top-bar branch changes
    void status;
    void branchID;
    void page;
    void load();
  });
</script>

<PageHeader
  title="Cashier shifts"
  icon="clock"
  sub="Drawer sessions, blind counts and cash variance across tills."
/>

<div class="cards">
  <StatCard label="Open now" value={String(summary?.open_count ?? 0)} icon="clock" tone="blue" />
  <StatCard
    label="Expected in open drawers"
    value={formatMoney(summary?.expected_open ?? '0')}
    icon="drawer"
    tone="blue"
  />
  <StatCard
    label="Variance today"
    value={formatMoney(summary?.variance_today ?? '0')}
    icon="scale"
    tone={Number(summary?.variance_today ?? 0) === 0 ? 'green' : 'red'}
  />
  <StatCard
    label="Closed today"
    value={String(summary?.closed_today ?? 0)}
    icon="check-circle"
    tone="green"
  />
</div>

<div class="filters">
  <Select bind:value={status} onchange={() => (page = 1)}>
    <option value="">All statuses</option>
    <option value="open">Open</option>
    <option value="closed">Closed</option>
  </Select>
  <Select bind:value={branchID} onchange={() => (page = 1)}>
    <option value="">All branches</option>
    {#each branches as b (b.id)}
      <option value={String(b.id)}>{b.name}</option>
    {/each}
  </Select>
</div>

<DataTable
  {columns}
  {rows}
  rowKey={(s) => s.id}
  {loading}
  empty="No shifts match — tills open one with a float from More → Open shift."
  onrowclick={(s) => (detail = s)}
>
  {#snippet cell(s: Shift, col)}
    {#if col.key === 'opened'}{formatDate(s.opened_at)}
    {:else if col.key === 'till'}<strong>{s.terminal}</strong>
    {:else if col.key === 'cashier'}{s.opened_by_name}
    {:else if col.key === 'branch'}{s.branch}
    {:else if col.key === 'float'}{formatMoney(s.float)}
    {:else if col.key === 'expected'}{s.expected !== null ? formatMoney(s.expected) : '—'}
    {:else if col.key === 'variance'}
      {#if s.variance !== null}
        <Badge tone={varianceTone(s)}>
          {Number(s.variance) >= 0 ? '+' : ''}{formatMoney(s.variance)}
        </Badge>
        {#if s.variance_status !== 'pending'}
          <Badge tone={s.variance_status === 'approved' ? 'success' : 'warning'}>
            {s.variance_status}
          </Badge>
        {/if}
      {:else}—{/if}
    {:else if col.key === 'status'}
      <Badge tone={s.status === 'open' ? 'success' : 'neutral'}>{s.status}</Badge>
    {/if}
  {/snippet}
</DataTable>

<Pager {page} {total} perPage={25} onpage={(p) => (page = p)} />

{#if detail}
  <ShiftDetail shift={detail} onclose={() => (detail = null)} onchanged={load} />
{/if}

<style>
  .cards {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
    gap: var(--space-3);
    margin-bottom: var(--space-4);
  }
  .filters {
    display: flex;
    gap: var(--space-2);
    margin-bottom: var(--space-3);
  }
</style>
