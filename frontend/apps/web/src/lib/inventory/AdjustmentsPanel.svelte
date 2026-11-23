<script lang="ts">
  import { ApiError } from '@mauzo/api-client';
  import type { Adjustment, Branch, Product } from '@mauzo/types';
  import { Badge, Button, DataTable, Modal, toast } from '@mauzo/ui';
  import { api } from '#lib/api.ts';
  import { auth } from '#lib/auth.svelte.ts';
  import LinePicker from '#lib/inventory/LinePicker.svelte';

  interface Props {
    branches: Branch[];
    reload: number;
  }
  let { branches, reload }: Props = $props();

  const REASONS = ['damage', 'expired', 'spoiled', 'count', 'other'];
  const columns = [
    { key: 'created_at', label: 'Created', width: '130px' },
    { key: 'branch_name', label: 'Branch', width: '130px' },
    { key: 'reason', label: 'Reason', width: '100px' },
    { key: 'items', label: 'Items' },
    { key: 'status', label: 'Status', width: '100px' },
    { key: 'actions', label: '', width: '220px', align: 'right' as const },
  ];

  let rows = $state<Adjustment[]>([]);
  let products = $state<Product[]>([]);
  let loading = $state(true);
  let open = $state(false);
  let branchId = $state('');
  let reason = $state('damage');
  let note = $state('');
  let lines = $state<{ product_id: string; qty: string }[]>([{ product_id: '', qty: '' }]);
  let busy = $state<string | null>(null);

  const canCreate = $derived(auth.can('inventory.adjustment.create'));
  const canApprove = $derived(auth.can('inventory.adjustment.approve'));

  async function load() {
    loading = true;
    try {
      const res = await api.list<Adjustment>('/api/v1/t/inventory/adjustments', { per_page: 50 });
      rows = res.data;
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Failed to load adjustments');
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    void reload;
    load();
  });

  async function ensureProducts() {
    if (products.length === 0) {
      const res = await api.list<Product>('/api/v1/t/catalog/products', { per_page: 200 });
      products = res.data;
    }
  }

  async function create() {
    const items = lines
      .filter((l) => l.product_id && l.qty)
      .map((l) => ({ product_id: l.product_id, qty: l.qty }));
    if (!branchId || items.length === 0) {
      toast.error('Branch and at least one item are required');
      return;
    }
    busy = 'create';
    try {
      await api.post('/api/v1/t/inventory/adjustments', {
        branch_id: branchId,
        reason,
        note,
        items,
      });
      toast.success('Adjustment drafted');
      open = false;
      lines = [{ product_id: '', qty: '' }];
      note = '';
      load();
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Failed');
    } finally {
      busy = null;
    }
  }

  async function act(row: Adjustment, verb: 'approve' | 'reject' | 'post') {
    busy = row.id + verb;
    try {
      await api.post(`/api/v1/t/inventory/adjustments/${row.id}/${verb}`);
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
  {#if canCreate}
    <Button
      size="sm"
      onclick={async () => {
        await ensureProducts();
        branchId = branches[0]?.id ?? '';
        open = true;
      }}
    >
      New adjustment
    </Button>
  {/if}
</div>

<DataTable {columns} {rows} rowKey={(r) => r.id} {loading} empty="No adjustments yet.">
  {#snippet cell(row, col)}
    {#if col.key === 'created_at'}
      {fmt(row.created_at)}
    {:else if col.key === 'items'}
      {row.items.map((i) => `${i.product_name} ${i.qty}`).join(', ')}
    {:else if col.key === 'status'}
      <Badge tone={row.status === 'posted' ? 'success' : row.status === 'rejected' ? 'danger' : 'neutral'}>
        {row.status}
      </Badge>
    {:else if col.key === 'actions'}
      {#if row.status === 'draft' && canApprove}
        <Button size="sm" variant="ghost" disabled={busy !== null} onclick={() => act(row, 'approve')}>
          Approve
        </Button>
        <Button size="sm" variant="ghost" disabled={busy !== null} onclick={() => act(row, 'reject')}>
          Reject
        </Button>
      {:else if row.status === 'approved' && canApprove}
        <Button size="sm" disabled={busy !== null} onclick={() => act(row, 'post')}>Post</Button>
      {/if}
    {:else}
      {(row as unknown as Record<string, unknown>)[col.key] ?? '—'}
    {/if}
  {/snippet}
</DataTable>

<Modal bind:open title="New stock adjustment">
  <label class="f">
    <span>Branch</span>
    <select bind:value={branchId}>
      {#each branches as b (b.id)}<option value={b.id}>{b.name}</option>{/each}
    </select>
  </label>
  <label class="f">
    <span>Reason</span>
    <select bind:value={reason}>
      {#each REASONS as r (r)}<option value={r}>{r}</option>{/each}
    </select>
  </label>
  <label class="f">
    <span>Note</span>
    <input bind:value={note} placeholder="Optional" />
  </label>
  <p class="hint">Quantities are signed deltas — negative reduces stock.</p>
  <LinePicker bind:lines {products} qtyPlaceholder="+5 or -5" />
  {#snippet footer()}
    <Button variant="ghost" onclick={() => (open = false)}>Cancel</Button>
    <Button onclick={create} disabled={busy === 'create'}>Create</Button>
  {/snippet}
</Modal>

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
  .hint {
    font-size: 12px;
    color: var(--fg-muted, #666);
    margin: 0 0 8px;
  }
</style>
