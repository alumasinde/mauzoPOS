<script lang="ts">
  import { ApiError } from '@mauzo/api-client';
  import type { Branch, Product, Transfer } from '@mauzo/types';
  import { Badge, Button, DataTable, Modal, toast } from '@mauzo/ui';
  import { api } from '#lib/api.ts';
  import { auth } from '#lib/auth.svelte.ts';
  import LinePicker from '#lib/inventory/LinePicker.svelte';

  interface Props {
    branches: Branch[];
    reload: number;
  }
  let { branches, reload }: Props = $props();

  const columns = [
    { key: 'created_at', label: 'Created', width: '120px' },
    { key: 'route', label: 'Route', width: '220px' },
    { key: 'items', label: 'Items' },
    { key: 'status', label: 'Status', width: '100px' },
    { key: 'actions', label: '', width: '210px', align: 'right' as const },
  ];

  let rows = $state<Transfer[]>([]);
  let products = $state<Product[]>([]);
  let loading = $state(true);
  let open = $state(false);
  let fromId = $state('');
  let toId = $state('');
  let note = $state('');
  let lines = $state<{ product_id: string; qty: string }[]>([{ product_id: '', qty: '' }]);
  let busy = $state<string | null>(null);

  const canCreate = $derived(auth.can('inventory.transfer.create'));
  const canReceive = $derived(auth.can('inventory.transfer.receive'));

  async function load() {
    loading = true;
    try {
      const res = await api.list<Transfer>('/api/v1/t/inventory/transfers', { per_page: 50 });
      rows = res.data;
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Failed to load transfers');
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
    if (!fromId || !toId || fromId === toId || items.length === 0) {
      toast.error('Choose two different branches and at least one item');
      return;
    }
    busy = 'create';
    try {
      await api.post('/api/v1/t/inventory/transfers', {
        from_branch_id: fromId,
        to_branch_id: toId,
        note,
        items,
      });
      toast.success('Transfer drafted');
      open = false;
      lines = [{ product_id: '', qty: '' }];
      load();
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Failed');
    } finally {
      busy = null;
    }
  }

  async function act(row: Transfer, verb: 'send' | 'receive' | 'cancel') {
    busy = row.id + verb;
    try {
      await api.post(
        `/api/v1/t/inventory/transfers/${row.id}/${verb}`,
        verb === 'receive' ? { items: [] } : undefined,
      );
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
        fromId = branches[0]?.id ?? '';
        toId = branches[1]?.id ?? '';
        open = true;
      }}
    >
      New transfer
    </Button>
  {/if}
</div>

<DataTable {columns} {rows} rowKey={(r) => r.id} {loading} empty="No transfers yet.">
  {#snippet cell(row, col)}
    {#if col.key === 'created_at'}
      {fmt(row.created_at)}
    {:else if col.key === 'route'}
      {row.from_name} → {row.to_name}
    {:else if col.key === 'items'}
      {row.items.map((i) => `${i.product_name} ×${i.qty_sent}`).join(', ')}
    {:else if col.key === 'status'}
      <Badge tone={row.status === 'received' ? 'success' : row.status === 'cancelled' ? 'danger' : 'info'}>
        {row.status}
      </Badge>
    {:else if col.key === 'actions'}
      {#if row.status === 'draft' && canCreate}
        <Button size="sm" disabled={busy !== null} onclick={() => act(row, 'send')}>Send</Button>
        <Button size="sm" variant="ghost" disabled={busy !== null} onclick={() => act(row, 'cancel')}>
          Cancel
        </Button>
      {:else if row.status === 'sent' && canReceive}
        <Button size="sm" disabled={busy !== null} onclick={() => act(row, 'receive')}>Receive</Button>
      {/if}
    {:else}
      {(row as unknown as Record<string, unknown>)[col.key] ?? '—'}
    {/if}
  {/snippet}
</DataTable>

<Modal bind:open title="New branch transfer">
  <div class="grid2">
    <label class="f">
      <span>From branch</span>
      <select bind:value={fromId}>
        {#each branches as b (b.id)}<option value={b.id}>{b.name}</option>{/each}
      </select>
    </label>
    <label class="f">
      <span>To branch</span>
      <select bind:value={toId}>
        {#each branches as b (b.id)}<option value={b.id}>{b.name}</option>{/each}
      </select>
    </label>
  </div>
  <label class="f">
    <span>Note</span>
    <input bind:value={note} placeholder="Optional" />
  </label>
  <LinePicker bind:lines {products} />
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
  .grid2 {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 10px;
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
</style>
