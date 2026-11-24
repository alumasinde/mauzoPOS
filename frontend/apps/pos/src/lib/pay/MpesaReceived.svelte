<script lang="ts">
  // Payments that just landed on this branch's till number. The one matching
  // the balance is highlighted so the cashier confirms with a single tap.
  import type { MpesaTxn } from '../desktop';
  import { formatMoney } from '../money';
  import { mpesafeed } from '../mpesafeed.svelte';
  import { type Tender, toCents } from './tender.svelte';

  let { tender, currency, onuse }: { tender: Tender; currency: string; onuse: (t: MpesaTxn) => void } =
    $props();

  const due = $derived(tender.remaining);
  // A payment "matches" when it covers the balance with up to KSh 100 extra
  // (customers round up); matches are listed first.
  const isMatch = (t: MpesaTxn) => {
    const c = toCents(t.amount);
    return c >= due && c - due <= 10000;
  };
  const rows = $derived(
    mpesafeed.items
      .filter((t) => t.trans_id && !tender.hasReference(t.trans_id))
      .sort((a, b) => Number(isMatch(b)) - Number(isMatch(a))),
  );

  function when(t: MpesaTxn): string {
    const d = new Date(t.trans_time || t.created_at);
    const mins = Math.round((Date.now() - d.getTime()) / 60000);
    if (mins < 1) return 'just now';
    if (mins < 60) return `${mins} min ago`;
    return d.toLocaleTimeString('en-KE', { hour: '2-digit', minute: '2-digit' });
  }
</script>

<section>
  <header>
    <h3>Received at this till</h3>
    <span class="live"><i></i>Live</span>
  </header>
  {#if rows.length === 0}
    <p class="empty">
      Waiting for payments… Ask the customer to pay to your till number — it appears here within seconds.
    </p>
  {:else}
    <ul>
      {#each rows as t (t.public_id)}
        <li>
          <button class:match={isMatch(t)} onclick={() => onuse(t)}>
            <span class="amt">{currency} {formatMoney(t.amount)}</span>
            <span class="who">
              <b>{t.payer_name || 'M-Pesa customer'}</b>
              <small>{t.phone} · {t.trans_id} · {when(t)}</small>
            </span>
            <span class="use">{isMatch(t) ? 'Use ✓' : 'Use'}</span>
          </button>
        </li>
      {/each}
    </ul>
  {/if}
</section>

<style>
  header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 8px;
  }
  h3 {
    margin: 0;
    font-size: 1.05rem;
  }
  .live {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    font-size: var(--text-sm);
    font-weight: 700;
    color: var(--color-success);
  }
  .live i {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: currentColor;
    animation: pulse 1.6s infinite;
  }
  @keyframes pulse {
    50% {
      opacity: 0.25;
    }
  }
  .empty {
    margin: 0;
    padding: 18px;
    text-align: center;
    color: var(--color-text-muted);
    border: 1px dashed var(--color-border);
    border-radius: var(--radius-md);
  }
  ul {
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
    gap: 8px;
    max-height: 290px;
    overflow-y: auto;
  }
  li button {
    width: 100%;
    display: grid;
    grid-template-columns: auto 1fr auto;
    align-items: center;
    gap: 14px;
    padding: 12px 14px;
    font: inherit;
    text-align: left;
    color: var(--color-text);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    cursor: pointer;
  }
  li button.match {
    border: 2px solid var(--color-success);
    background: color-mix(in srgb, var(--color-success) 10%, var(--color-surface));
  }
  .amt {
    font-size: 1.4rem;
    font-weight: 800;
    font-variant-numeric: tabular-nums;
  }
  .who {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }
  .who small {
    color: var(--color-text-muted);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .use {
    font-weight: 800;
    color: var(--color-success);
  }
</style>
