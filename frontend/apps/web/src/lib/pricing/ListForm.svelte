<script lang="ts">
  import { ApiError } from '@mauzo/api-client';
  import type { Branch, PriceList, Product } from '@mauzo/types';
  import { Button, Modal, TextField, toast } from '@mauzo/ui';
  import { api } from '#lib/api.ts';

  interface Item {
    product_id: string;
    min_qty: string;
    price: string;
  }
  interface Props {
    branches: Branch[];
    target: PriceList | null;
    open: boolean;
    onclose: () => void;
    ondone: () => void;
  }
  let { branches, target, open = $bindable(), onclose, ondone }: Props = $props();

  let name = $state('');
  let branchId = $state('');
  let group = $state('');
  let priority = $state('0');
  let validFrom = $state('');
  let validTo = $state('');
  let items = $state<Item[]>([]);
  let products = $state<Product[]>([]);
  let saving = $state(false);

  async function ensureProducts() {
    if (products.length === 0) {
      try {
        const res = await api.list<Product>('/api/v1/t/catalog/products', { per_page: 200 });
        products = res.data;
      } catch {
        /* picker best-effort */
      }
    }
  }

  $effect(() => {
    if (open) {
      ensureProducts();
      name = target?.name ?? '';
      branchId = target?.branch_id ?? '';
      group = target?.customer_group ?? '';
      priority = String(target?.priority ?? 0);
      validFrom = target?.valid_from?.slice(0, 16) ?? '';
      validTo = target?.valid_to?.slice(0, 16) ?? '';
      items =
        target?.items?.map((i) => ({
          product_id: i.product_id,
          min_qty: i.min_qty,
          price: i.price,
        })) ?? [];
    }
  });

  async function save() {
    if (!name.trim()) {
      toast.error('Name is required');
      return;
    }
    const rows = items
      .filter((i) => i.product_id && i.price)
      .map((i) => ({
        product_id: i.product_id,
        ...(i.min_qty ? { min_qty: i.min_qty } : {}),
        price: i.price,
      }));
    saving = true;
    try {
      const body = {
        name: name.trim(),
        branch_id: branchId || null,
        customer_group: group || null,
        priority: Number(priority) || 0,
        valid_from: validFrom ? new Date(validFrom).toISOString() : null,
        valid_to: validTo ? new Date(validTo).toISOString() : null,
        items: rows,
      };
      if (target) await api.patch(`/api/v1/t/pricing/lists/${target.id}`, body);
      else await api.post('/api/v1/t/pricing/lists', body);
      toast.success(target ? 'List updated' : 'List created');
      ondone();
      onclose();
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Save failed');
    } finally {
      saving = false;
    }
  }
</script>

<Modal bind:open title={target ? 'Edit price list' : 'New price list'} size="lg" {onclose}>
  <div class="grid3">
    <TextField label="Name" bind:value={name} />
    <label class="f">
      <span>Branch</span>
      <select bind:value={branchId}>
        <option value="">All branches</option>
        {#each branches as b (b.id)}<option value={b.id}>{b.name}</option>{/each}
      </select>
    </label>
    <TextField label="Customer group (blank = everyone)" bind:value={group} />
    <TextField label="Priority (higher wins)" type="number" bind:value={priority} />
    <label class="f">
      <span>Valid from</span>
      <input type="datetime-local" bind:value={validFrom} />
    </label>
    <label class="f">
      <span>Valid to</span>
      <input type="datetime-local" bind:value={validTo} />
    </label>
  </div>

  <h4>Items &amp; tiers</h4>
  <p class="hint">
    Multiple rows per product create quantity tiers — the largest min qty the sale reaches applies.
  </p>
  <div class="items">
    {#each items as it, i (i)}
      <div class="irow">
        <select bind:value={it.product_id}>
          <option value="">Product…</option>
          {#each products as p (p.id)}<option value={p.id}>{p.name} ({p.sku})</option>{/each}
        </select>
        <input type="number" min="0" step="any" placeholder="Min qty" bind:value={it.min_qty} />
        <input type="number" min="0" step="any" placeholder="Price" bind:value={it.price} />
        <Button variant="ghost" size="sm" onclick={() => (items = items.filter((_, j) => j !== i))}>✕</Button>
      </div>
    {/each}
    <Button
      variant="ghost"
      size="sm"
      onclick={() => (items = [...items, { product_id: '', min_qty: '1', price: '' }])}
    >
      + Add item
    </Button>
  </div>

  {#snippet footer()}
    <Button variant="ghost" onclick={onclose}>Cancel</Button>
    <Button onclick={save} disabled={saving}>{saving ? 'Saving…' : 'Save'}</Button>
  {/snippet}
</Modal>

<style>
  .grid3 {
    display: grid;
    grid-template-columns: 1fr 1fr 1fr;
    gap: 10px;
  }
  .f {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .f span {
    font-size: 12px;
    color: var(--fg-muted, #666);
  }
  .f select,
  .f input,
  .irow select,
  .irow input {
    padding: 6px 8px;
    border: 1px solid var(--border, #ddd);
    border-radius: 6px;
    background: var(--bg, #fff);
    color: var(--fg, #222);
    min-width: 0;
  }
  h4 {
    margin: 14px 0 4px;
  }
  .hint {
    font-size: 12px;
    color: var(--fg-muted, #666);
    margin: 0 0 8px;
  }
  .items {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .irow {
    display: grid;
    grid-template-columns: 2fr 90px 100px 32px;
    gap: 6px;
    align-items: center;
  }
</style>
