<script lang="ts">
  import type { Product } from '@mauzo/types';
  import { Button } from '@mauzo/ui';

  interface Line {
    product_id: string;
    qty: string;
  }
  interface Props {
    products: Product[];
    lines: Line[];
    qtyLabel?: string;
    qtyPlaceholder?: string;
  }
  let { products, lines = $bindable(), qtyLabel = 'Qty', qtyPlaceholder = 'Qty' }: Props = $props();
</script>

<div class="lines">
  {#each lines as l, i (i)}
    <div class="line">
      <select bind:value={l.product_id}>
        <option value="">Product…</option>
        {#each products as p (p.id)}<option value={p.id}>{p.name} ({p.sku})</option>{/each}
      </select>
      <input type="number" step="any" placeholder={qtyPlaceholder} title={qtyLabel} bind:value={l.qty} />
      <Button variant="ghost" size="sm" onclick={() => (lines = lines.filter((_, j) => j !== i))}>✕</Button>
    </div>
  {/each}
  <Button variant="ghost" size="sm" onclick={() => (lines = [...lines, { product_id: '', qty: '' }])}>
    + Add line
  </Button>
</div>

<style>
  .lines {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .line {
    display: grid;
    grid-template-columns: 1fr 100px 32px;
    gap: 6px;
    align-items: center;
  }
  .line select,
  .line input {
    padding: 6px 8px;
    border: 1px solid var(--border, #ddd);
    border-radius: 6px;
    background: var(--bg, #fff);
    color: var(--fg, #222);
    min-width: 0;
  }
</style>
