<script lang="ts">
  import { ApiError } from '@mauzo/api-client';
  import type { Branch, StockCount } from '@mauzo/types';
  import { Badge, Button, DataTable, Modal, toast } from '@mauzo/ui';
  import { api } from '#lib/api.ts';
  import { auth } from '#lib/auth.svelte.ts';

  interface Props {
    branches: Branch[];
    reload: number;
  }
  let { branches, reload }: Props = $props();

  const columns = [
    { key: 'created_at', label: 'Created', width: '120px' },
    { key: 'branch_name', label: 'Branch', width: '140px' },
    { key: 'items', label: 'Lines', width: '80px', align: 'right' as const },
    { key: 'status', label: 'Status', width: '100px' },
    { key: 'actions', label: '', width: '200px', align: 'right' as const },
  ];

  let rows = $state<StockCount[]>([]);
  let loading = $state(true);
  let open = $state(false);
  let branchId = $state('');
  let note = $state('');
  let busy = $state<string | null>(null);

  // Count sheet state
  let sheet = $state<StockCount | null>(null);
  let counted = $state<Record<string, string>>({});

  const canManage = $derived(auth.can('inventory.stocktake.manage'));

  async function load() {
    loading = true;
    try {
      const res = await api.list<StockCount>('/api/v1/t/inventory/counts', { per_page: 50 });
      rows = res.data;
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Failed to load counts');
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    void reload;
    load();
  });

  async function create() {
    if (!branchId) {
      toast.error('Choose a branch');
      return;
    }
    busy = 'create';
    try {
      const c = await api.post<StockCount>('/api/v1/t/inventory/counts', {
        branch_id: branchId,
        note,
      });
      toast.success('Count opened');
      open = false;
      load();
      if (c) openSheet(c.id);
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Failed');
    } finally {
      busy = null;
    }
  }

  async function openSheet(id: string) {
    try {
      sheet = await api.get<StockCount>(`/api/v1/t/inventory/counts/${id}`);
      counted = Object.fromEntries(sheet!.items.map((i) => [i.id!, i.counted_qty ?? i.expected_qty ?? '']));
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Failed to load count');
    }
  }

  async function saveCounts() {
    if (!sheet) return;
    busy = 'sheet';
    try {
      sheet = await api.put<StockCount>(`/api/v1/t/inventory/counts/${sheet.id}/items`, {
        counts: counted,
      });
      toast.success('Counts saved');
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Failed');
    } finally {
      busy = null;
    }
  }

  async function postCount() {
    if (!sheet) return;
    busy = 'post';
    try {
      await saveCountsQuiet();
      await api.post(`/api/v1/t/inventory/counts/${sheet.id}/post`);
      toast.success('Count posted — variances applied');
      sheet = null;
      load();
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Failed');
    } finally {
      busy = null;
    }
  }

  async function saveCountsQuiet() {
    if (!sheet) return;
    sheet = await api.put<StockCount>(`/api/v1/t/inventory/counts/${sheet.id}/items`, {
      counts: counted,
    });
  }

  async function cancelCount(row: StockCount) {
    busy = row.id;
    try {
      await api.post(`/api/v1/t/inventory/counts/${row.id}/cancel`);
      load();
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Failed');
    } finally {
      busy = null;
    }
  }

  const fmt = (s?: string | null) => (s ? new Date(s).toLocaleDateString() : '—');
</script>

<div class="bar">
  {#if canManage}
    <Button
      size="sm"
      onclick={() => {
        branchId = branches[0]?.id ?? '';
        open = true;
      }}
    >
      New stocktake
    </Button>
  {/if}
</div>

<DataTable {columns} {rows} rowKey={(r) => r.id} {loading} empty="No stocktakes yet.">
  {#snippet cell(row, col)}
    {#if col.key === 'created_at'}
      {fmt(row.created_at)}
    {:else if col.key === 'items'}
      {row.items.length}
    {:else if col.key === 'status'}
      <Badge tone={row.status === 'posted' ? 'success' : row.status === 'cancelled' ? 'danger' : 'info'}>
        {row.status}
      </Badge>
    {:else if col.key === 'actions'}
      {#if row.status === 'open' && canManage}
        <Button size="sm" disabled={busy !== null} onclick={() => openSheet(row.id)}>Count</Button>
        <Button size="sm" variant="ghost" disabled={busy !== null} onclick={() => cancelCount(row)}>
          Cancel
        </Button>
      {/if}
    {:else}
      {(row as unknown as Record<string, unknown>)[col.key] ?? '—'}
    {/if}
  {/snippet}
</DataTable>

<Modal bind:open title="New stocktake">
  <label class="f">
    <span>Branch</span>
    <select bind:value={branchId}>
      {#each branches as b (b.id)}<option value={b.id}>{b.name}</option>{/each}
    </select>
  </label>
  <label class="f">
    <span>Note</span>
    <input bind:value={note} placeholder="Optional" />
  </label>
  {#snippet footer()}
    <Button variant="ghost" onclick={() => (open = false)}>Cancel</Button>
    <Button onclick={create} disabled={busy === 'create'}>Open count</Button>
  {/snippet}
</Modal>

{#if sheet}
  <Modal open={true} title={`Count — ${sheet.branch_name}`} size="lg" onclose={() => (sheet = null)}>
    <div class="sheet">
      <div class="srow head">
        <span>Product</span><span>Expected</span><span>Counted</span><span>Variance</span>
      </div>
      {#each sheet.items as it (it.id)}
        <div class="srow">
          <span>{it.product_name}{it.variant_name ? ` · ${it.variant_name}` : ''}</span>
          <span>{it.expected_qty}</span>
          <input type="number" step="any" bind:value={counted[it.id!]} disabled={sheet.status !== 'open'} />
          <span
            class:neg={counted[it.id!] !== undefined && Number(counted[it.id!]) - Number(it.expected_qty) < 0}
          >
            {counted[it.id!] !== undefined && counted[it.id!] !== ''
              ? (Number(counted[it.id!]) - Number(it.expected_qty)).toString()
              : '—'}
          </span>
        </div>
      {/each}
    </div>
    {#snippet footer()}
      {#if sheet?.status === 'open' && canManage}
        <Button variant="ghost" onclick={() => (sheet = null)}>Close</Button>
        <Button variant="ghost" onclick={saveCounts} disabled={busy === 'sheet'}>Save counts</Button>
        <Button onclick={postCount} disabled={busy !== null}>Post variances</Button>
      {/if}
    {/snippet}
  </Modal>
{/if}

<style>
  .bar {
    display: flex;
    justify-content: flex-end;
    margin-bottom: 10px;
  }
  .f {
    display: flex;
    flex-direction: column;
    gap: 4px;
    margin-bottom: 10px;
  }
  .f span {
    font-size: 12px;
    color: var(--fg-muted, #666);
  }
  .f select,
  .f input {
    padding: 6px 8px;
    border: 1px solid var(--border, #ddd);
    border-radius: 6px;
    background: var(--bg, #fff);
    color: var(--fg, #222);
  }
  .sheet {
    display: flex;
    flex-direction: column;
    gap: 4px;
    max-height: 50vh;
    overflow-y: auto;
  }
  .srow {
    display: grid;
    grid-template-columns: 2fr 90px 110px 90px;
    gap: 8px;
    align-items: center;
    font-variant-numeric: tabular-nums;
  }
  .srow.head {
    font-size: 12px;
    color: var(--fg-muted, #666);
    font-weight: 600;
  }
  .srow input {
    padding: 5px 8px;
    border: 1px solid var(--border, #ddd);
    border-radius: 6px;
    background: var(--bg, #fff);
    color: var(--fg, #222);
  }
  .neg {
    color: var(--danger, #dc2626);
  }
</style>
