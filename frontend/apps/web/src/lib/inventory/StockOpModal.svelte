<script lang="ts">
  import { ApiError } from '@mauzo/api-client';
  import type { Branch, Product, StockLineRequest, StockLocation } from '@mauzo/types';
  import { Button, Modal, toast } from '@mauzo/ui';
  import { api } from '#lib/api.ts';

  interface Props {
    kind: 'receive' | 'wastage';
    branches: Branch[];
    open: boolean;
    onclose?: () => void;
    ondone: () => void;
  }
  let { kind, branches, open = $bindable(), onclose, ondone }: Props = $props();

  const REASONS = ['damage', 'expired', 'spoiled', 'staff_meal', 'sample', 'other'];

  let branchId = $state('');
  let reason = $state('damage');
  let note = $state('');
  let lines = $state<StockLineRequest[]>([{ product_id: '', qty: '' }]);
  let locations = $state<StockLocation[]>([]);
  let products = $state<Product[]>([]);
  let saving = $state(false);

  const title = $derived(kind === 'receive' ? 'Receive stock' : 'Record wastage');

  async function loadData() {
    products = [];
    try {
      const res = await api.list<Product>('/api/v1/t/catalog/products', { per_page: 200 });
      products = res.data;
    } catch {
      /* catalog list is best-effort for the picker */
    }
    if (branchId) {
      try {
        locations = await api.get<StockLocation[]>(`/api/v1/t/inventory/locations?branch_id=${branchId}`);
      } catch {
        locations = [];
      }
    }
  }

  $effect(() => {
    if (open) {
      branchId = branches[0]?.id ?? '';
      lines = [{ product_id: '', qty: '' }];
      reason = 'damage';
      note = '';
      loadData();
    }
  });

  $effect(() => {
    if (branchId) loadData();
  });

  function addLine() {
    lines = [...lines, { product_id: '', qty: '' }];
  }

  function removeLine(i: number) {
    lines = lines.filter((_, j) => j !== i);
  }

  async function save() {
    if (!branchId) {
      toast.error('Choose a branch');
      return;
    }
    const items = lines
      .filter((l) => l.product_id && l.qty)
      .map((l) => ({
        ...l,
        ...(l.variant_id ? {} : { variant_id: undefined }),
        ...(l.location_id ? {} : { location_id: undefined }),
        ...(l.batch_no ? {} : { batch_no: undefined }),
        ...(l.expiry ? {} : { expiry: undefined }),
        ...(l.unit_cost ? {} : { unit_cost: undefined }),
      }));
    if (items.length === 0) {
      toast.error('Add at least one item');
      return;
    }
    saving = true;
    try {
      if (kind === 'receive') {
        await api.post('/api/v1/t/inventory/receive', {
          branch_id: branchId,
          note,
          items,
        });
        toast.success('Stock received');
      } else {
        await api.post('/api/v1/t/inventory/wastage', {
          branch_id: branchId,
          reason,
          note,
          items,
        });
        toast.success('Wastage recorded');
      }
      ondone();
      onclose?.();
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Operation failed');
    } finally {
      saving = false;
    }
  }
</script>

<Modal bind:open {title} onclose={() => onclose?.()} size="lg">
  <div class="grid2">
    <label class="f">
      <span>Branch</span>
      <select bind:value={branchId}>
        {#each branches as b (b.id)}<option value={b.id}>{b.name}</option>{/each}
      </select>
    </label>
    {#if kind === 'wastage'}
      <label class="f">
        <span>Reason</span>
        <select bind:value={reason}>
          {#each REASONS as r (r)}<option value={r}>{r}</option>{/each}
        </select>
      </label>
    {/if}
  </div>
  <label class="f">
    <span>Note</span>
    <input bind:value={note} placeholder="Optional" />
  </label>

  <div class="lines">
    {#each lines as l, i (i)}
      <div class="line">
        <select class="prod" bind:value={l.product_id}>
          <option value="">Product…</option>
          {#each products as p (p.id)}<option value={p.id}>{p.name} ({p.sku})</option>{/each}
        </select>
        <input class="qty" type="number" min="0" step="any" placeholder="Qty" bind:value={l.qty} />
        {#if kind === 'receive'}
          <input class="cost" type="number" min="0" step="any" placeholder="Cost" bind:value={l.unit_cost} />
          <input class="batch" placeholder="Batch #" bind:value={l.batch_no} />
          <input class="exp" type="date" bind:value={l.expiry} title="Expiry" />
        {/if}
        <select class="loc" bind:value={l.location_id} title="Location">
          <option value="">Default location</option>
          {#each locations as loc (loc.id)}<option value={loc.id}>{loc.name}</option>{/each}
        </select>
        <Button variant="ghost" size="sm" onclick={() => removeLine(i)}>✕</Button>
      </div>
    {/each}
    <Button variant="ghost" size="sm" onclick={addLine}>+ Add line</Button>
  </div>

  {#snippet footer()}
    <Button variant="ghost" onclick={() => onclose?.()}>Cancel</Button>
    <Button onclick={save} disabled={saving}>{saving ? 'Posting…' : title}</Button>
  {/snippet}
</Modal>

<style>
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
  .lines {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-top: 6px;
  }
  .line {
    display: grid;
    grid-template-columns: 2fr 70px;
    grid-auto-columns: minmax(0, 1fr);
    grid-auto-flow: column;
    gap: 6px;
    align-items: center;
  }
  .line input,
  .line select,
  .f select,
  .f input {
    padding: 6px 8px;
    border: 1px solid var(--border, #ddd);
    border-radius: 6px;
    background: var(--bg, #fff);
    color: var(--fg, #222);
    min-width: 0;
  }
</style>
