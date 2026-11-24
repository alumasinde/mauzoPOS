<script lang="ts">
  // Card (separate bank terminal), bank transfer and customer account. These
  // are charged exactly — no change.
  import { cart } from '../cart.svelte';
  import CustomerPick from '../components/CustomerPick.svelte';
  import { formatMoney } from '../money';
  import { type Method, type Tender, fromCents } from './tender.svelte';

  let { tender, currency, method }: { tender: Tender; currency: string; method: Method } = $props();

  let approval = $state('');
  let last4 = $state('');
  let reference = $state('');

  const ok = $derived(
    method === 'card'
      ? approval.trim().length >= 4 && (!last4 || /^\d{4}$/.test(last4))
      : method === 'bank'
        ? reference.trim().length >= 4
        : !!cart.customerID,
  );

  function take() {
    if (!ok) return;
    if (method === 'card') {
      const ref = (last4 ? `${last4}/` : '') + approval.trim().toUpperCase();
      tender.take('card', tender.remaining, ref, last4 ? `Card ••••${last4}` : 'Card');
    } else if (method === 'bank') {
      tender.take('bank', tender.remaining, reference.trim(), 'Bank transfer');
    } else {
      tender.take('credit', tender.remaining, '', cart.customerName);
    }
    approval = last4 = reference = '';
  }
</script>

<div class="other">
  <div class="due">
    <span>Charge</span>
    <strong>{currency} {formatMoney(fromCents(tender.remaining))}</strong>
  </div>

  {#if method === 'card'}
    <p class="hint">Run the card on the bank terminal first, then copy from its slip:</p>
    <div class="row">
      <label>
        <span>Approval / auth code *</span>
        <input bind:value={approval} maxlength="12" autocomplete="off" />
      </label>
      <label>
        <span>Last 4 digits</span>
        <input bind:value={last4} maxlength="4" inputmode="numeric" placeholder="1234" />
      </label>
    </div>
  {:else if method === 'bank'}
    <label>
      <span>Transfer reference *</span>
      <input bind:value={reference} maxlength="40" autocomplete="off" />
    </label>
  {:else}
    <p class="hint">Charge to the customer's account (credit limit is checked).</p>
    <CustomerPick />
  {/if}

  <button disabled={!ok} onclick={take}>Charge {currency} {formatMoney(fromCents(tender.remaining))}</button>
</div>

<style>
  .other {
    display: grid;
    gap: 14px;
  }
  .due {
    padding: 12px 16px;
    border-radius: var(--radius-md);
    background: var(--color-surface-muted);
  }
  .due span {
    display: block;
    font-size: var(--text-sm);
    color: var(--color-text-muted);
  }
  .due strong {
    font-size: 2rem;
    font-variant-numeric: tabular-nums;
  }
  .hint {
    margin: 0;
    color: var(--color-text-muted);
  }
  .row {
    display: grid;
    grid-template-columns: 1.5fr 1fr;
    gap: 10px;
  }
  label span {
    display: block;
    font-size: var(--text-sm);
    color: var(--color-text-muted);
    margin-bottom: 4px;
  }
  input {
    width: 100%;
    height: 56px;
    padding: 0 14px;
    font: inherit;
    font-size: 1.4rem;
    font-weight: 700;
    color: var(--color-text);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
  }
  button {
    height: 60px;
    font: inherit;
    font-size: 1.15rem;
    font-weight: 800;
    color: #fff;
    background: var(--color-primary);
    border: 0;
    border-radius: var(--radius-md);
    cursor: pointer;
  }
  button:disabled {
    opacity: 0.45;
    cursor: default;
  }
</style>
