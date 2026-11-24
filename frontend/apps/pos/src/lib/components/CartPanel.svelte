<script lang="ts">
  import { cart } from '../cart.svelte';
  import { formatMoney } from '../money';
  import { pulse } from '../pulse.svelte';
  import PaidDone from './PaidDone.svelte';

  let { currency = 'KSh' }: { currency?: string } = $props();

  const money = (s: string | number) => formatMoney(String(s), currency);

  // The next item scanned starts a new sale: the last sale's card goes.
  $effect(() => {
    if (cart.count > 0 && pulse.lastSale) pulse.lastSale = null;
  });
</script>

<aside class="cart">
  <div class="head rowhead">
    <span class="n">#</span><span class="item">Item</span>
    <span class="qty">Qty</span><span class="num">Price</span><span class="num">Total</span>
  </div>
  <div class="lines">
    {#each cart.rows as row, i (i)}
      <div
        class="row"
        class:sel={cart.selected === i}
        role="button"
        tabindex="0"
        onclick={() => cart.select(i)}
        onkeydown={(e) => e.key === 'Enter' && cart.select(i)}
      >
        <span class="n">{i + 1}</span>
        <span class="item">
          <b>{row.name}</b>
          <small>
            {row.sku ? `SKU: ${row.sku}` : ''}{row.priced?.promo_name ? ` · ${row.priced.promo_name}` : ''}
          </small>
        </span>
        <span class="qty">
          <button
            class="step"
            onclick={(e) => {
              e.stopPropagation();
              cart.bump(i, -1);
            }}
          >
            −
          </button>
          <span class="qv">{row.qty}</span>
          <button
            class="step"
            onclick={(e) => {
              e.stopPropagation();
              cart.bump(i, 1);
            }}
          >
            +
          </button>
        </span>
        <span class="num">{money(row.priced?.unit_price ?? '—')}</span>
        <span class="num">{money(row.priced?.total ?? '—')}</span>
      </div>
    {:else}
      {#if pulse.lastSale}
        {#key pulse.lastSale}
          <PaidDone {currency} result={pulse.lastSale} ondone={() => (pulse.lastSale = null)} />
        {/key}
      {:else}
        <p class="empty">Scan a barcode or tap a product to start a sale.</p>
      {/if}
    {/each}
  </div>
  <div class="totals">
    <div><span>Subtotal</span><b>{money(cart.totals.subtotal)}</b></div>
    <div class:neg={cart.totals.discount !== '0.00'}>
      <span>Discount</span><b>- {money(cart.totals.discount)}</b>
    </div>
    <div><span>VAT (16%)</span><b>{money(cart.totals.vat)}</b></div>
  </div>
  <div class="grand">
    <span>Total</span>
    <b>{currency} {money(cart.totals.total)}</b>
  </div>
</aside>

<style>
  .cart {
    display: flex;
    flex-direction: column;
    background: var(--color-surface);
    border-right: 1px solid var(--color-border);
    min-height: 0;
  }
  .rowhead,
  .row {
    display: grid;
    grid-template-columns: 26px 1fr 96px 74px 82px;
    align-items: center;
    gap: var(--space-1);
    padding: var(--space-2) var(--space-3);
  }
  .head {
    font-size: var(--text-xs);
    font-weight: 700;
    text-transform: uppercase;
    color: var(--color-text-muted);
    border-bottom: 1px solid var(--color-border);
  }
  .lines {
    flex: 1;
    overflow-y: auto;
    min-height: 0;
  }
  .row {
    width: 100%;
    border: none;
    background: none;
    font: inherit;
    cursor: pointer;
    text-align: left;
    border-bottom: 1px solid var(--color-surface-muted);
  }
  .row.sel {
    background: color-mix(in srgb, var(--color-primary) 8%, transparent);
  }
  .n {
    color: var(--color-text-muted);
  }
  .item {
    min-width: 0;
  }
  .item b {
    display: block;
    font-weight: 600;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .item small {
    color: var(--color-text-muted);
    font-size: var(--text-xs);
  }
  .qty {
    display: inline-flex;
    align-items: center;
    gap: 2px;
  }
  .step {
    width: 26px;
    height: 26px;
    border-radius: var(--radius-sm);
    border: 1px solid var(--color-border);
    background: var(--color-surface);
    font-weight: 700;
    cursor: pointer;
  }
  .qv {
    min-width: 30px;
    text-align: center;
    font-weight: 600;
  }
  .num {
    text-align: right;
    font-variant-numeric: tabular-nums;
  }
  .empty {
    padding: var(--space-5);
    color: var(--color-text-muted);
  }
  .totals {
    border-top: 1px solid var(--color-border);
    padding: var(--space-2) var(--space-3);
  }
  .totals div {
    display: flex;
    justify-content: space-between;
    padding: 2px 0;
    font-variant-numeric: tabular-nums;
  }
  .totals .neg b {
    color: var(--color-danger);
  }
  .grand {
    display: flex;
    justify-content: space-between;
    align-items: center;
    background: var(--color-primary);
    color: #fff;
    padding: var(--space-3);
    font-size: var(--text-xl);
    font-weight: 700;
  }
</style>
