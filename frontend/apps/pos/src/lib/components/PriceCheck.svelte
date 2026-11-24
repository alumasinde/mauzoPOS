<script lang="ts">
  // Price check modal: scan/enter a barcode or SKU, show the sell price.
  import { untrack } from 'svelte';
  import { desktop } from '../desktop';
  import type { PriceCheckResult } from '../desktop';
  import { formatMoney } from '../money';

  let {
    code = '',
    currency = 'KSh',
    onclose,
  }: {
    code?: string;
    currency?: string;
    onclose: () => void;
  } = $props();

  const bridge = desktop();
  let value = $state(untrack(() => code));
  let result = $state<PriceCheckResult | null>(null);
  let busy = $state(false);

  async function run() {
    if (!bridge || !value.trim()) return;
    busy = true;
    result = await bridge.PriceCheck(value.trim()).catch(() => null);
    busy = false;
  }
</script>

<div class="modal">
  <h3>Price check</h3>
  <div class="pcRow">
    <input bind:value placeholder="Barcode or SKU" onkeydown={(e) => e.key === 'Enter' && run()} />
    <button class="go" onclick={run} disabled={busy}>{busy ? '…' : 'Check'}</button>
  </div>
  {#if result}
    {#if result.unknown}
      <p class="muted">No product matches <code>{result.code}</code>.</p>
    {:else}
      <div class="pcRes">
        <b>{result.name}</b>
        <span class="sku">{result.sku}</span>
        <span class="pcPrice">
          {currency}
          {formatMoney(result.price ?? '0')}{result.sell_by === 'weight' ? ' /kg' : ''}
        </span>
      </div>
    {/if}
  {/if}
  <button class="ghost" onclick={onclose}>Close</button>
</div>

<style>
  h3 {
    margin: 0 0 var(--space-3);
  }
  .muted {
    color: var(--color-text-muted);
  }
  input {
    width: 100%;
    padding: var(--space-3);
    font: inherit;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    box-sizing: border-box;
  }
  .pcRow {
    display: flex;
    gap: var(--space-2);
    margin-bottom: var(--space-3);
  }
  .pcRes {
    display: flex;
    flex-direction: column;
    gap: 2px;
    margin: var(--space-3) 0;
  }
  .pcRes .sku {
    color: var(--color-text-muted);
    font-size: var(--text-sm);
  }
  .pcPrice {
    font-size: var(--text-2xl);
    font-weight: 700;
    color: var(--color-primary);
  }
  .go {
    padding: var(--space-3) var(--space-4);
    font-weight: 700;
    color: #fff;
    background: var(--color-primary);
    border: none;
    border-radius: var(--radius-md);
    cursor: pointer;
  }
  .go:disabled {
    opacity: 0.5;
  }
  .ghost {
    margin-top: var(--space-3);
    padding: var(--space-2) var(--space-4);
    font: inherit;
    background: none;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    cursor: pointer;
  }
</style>
