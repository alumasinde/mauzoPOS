<script lang="ts">
  // Amount-due card at the top of the payment drawer: the one big number the
  // cashier needs right now (to pay → change), plus the payments taken so far.
  import { Icon } from '@mauzo/ui';
  import { cart } from '../cart.svelte';
  import { formatMoney } from '../money';
  import { LABELS, type Tender, fromCents } from './tender.svelte';

  let { tender, currency }: { tender: Tender; currency: string } = $props();

  const mode = $derived<'due' | 'change' | 'exact'>(
    tender.remaining > 0 ? 'due' : tender.change > 0 ? 'change' : 'exact',
  );
  const label = $derived(
    mode === 'due'
      ? tender.lines.length
        ? 'Still to pay'
        : 'Amount due'
      : mode === 'change'
        ? 'Give change'
        : 'Paid exactly',
  );
  const big = $derived(mode === 'due' ? tender.remaining : mode === 'change' ? tender.change : tender.total);
</script>

<div class="card {mode}">
  <div class="main">
    <span class="lbl">{label}</span>
    <strong>{currency} {formatMoney(fromCents(big))}</strong>
    {#if tender.lines.length}<small>Sale total {currency} {formatMoney(fromCents(tender.total))}</small>{/if}
  </div>
  <div class="meta">
    <span><Icon name="cart" size={18} /> Items: {cart.count}</span>
    <span><Icon name="receipt" size={18} /> VAT included</span>
  </div>
</div>

{#if tender.lines.length}
  <ul aria-label="Payments taken">
    {#each tender.lines as l, i (i)}
      <li>
        <span class="m">{LABELS[l.method]}</span>
        <span class="n">{l.note ?? ''}{l.reference && l.method === 'mpesa' ? ` · ${l.reference}` : ''}</span>
        <b>{formatMoney(fromCents(l.applied))}</b>
        <button aria-label="Remove {LABELS[l.method]} payment" onclick={() => tender.remove(i)}>
          <Icon name="x" size={16} />
        </button>
      </li>
    {/each}
  </ul>
{/if}

<style>
  .card {
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 16px 20px;
    border-radius: var(--radius-lg);
    background: color-mix(in srgb, var(--color-primary) 9%, var(--color-surface));
    border: 1px solid color-mix(in srgb, var(--color-primary) 22%, transparent);
  }
  .main {
    flex: 1;
    display: grid;
    min-width: 0;
  }
  .lbl {
    font-size: var(--text-sm);
    font-weight: 800;
    letter-spacing: 0.04em;
    text-transform: uppercase;
    color: var(--color-text-muted);
  }
  strong {
    font-size: 2.6rem;
    line-height: 1.1;
    font-variant-numeric: tabular-nums;
  }
  small {
    color: var(--color-text-muted);
  }
  .meta {
    display: grid;
    gap: 8px;
    padding-left: 16px;
    border-left: 1px solid var(--color-border);
    color: var(--color-text-muted);
    white-space: nowrap;
  }
  .meta span {
    display: flex;
    gap: 6px;
    align-items: center;
  }
  .change {
    background: var(--color-success);
    border-color: transparent;
  }
  .change .lbl,
  .change small,
  .change .meta {
    color: rgb(255 255 255 / 0.9);
  }
  .change strong {
    font-size: 3rem;
    color: #fff;
  }
  .exact {
    background: color-mix(in srgb, var(--color-success) 12%, var(--color-surface));
    border-color: color-mix(in srgb, var(--color-success) 35%, transparent);
  }
  .exact strong {
    color: var(--color-success);
  }
  ul {
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
    gap: 6px;
  }
  li {
    display: grid;
    grid-template-columns: auto 1fr auto auto;
    gap: 10px;
    align-items: center;
    padding: 8px 12px;
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
  }
  .m {
    font-weight: 800;
  }
  .n {
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
    color: var(--color-text-muted);
  }
  li b {
    font-size: 1.15rem;
    font-variant-numeric: tabular-nums;
  }
  li button {
    display: grid;
    place-items: center;
    width: 32px;
    height: 32px;
    color: var(--color-text-muted);
    background: var(--color-surface-muted);
    border: 0;
    border-radius: 50%;
    cursor: pointer;
  }
</style>
