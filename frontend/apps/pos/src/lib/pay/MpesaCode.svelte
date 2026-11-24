<script lang="ts">
  // Fallback: the customer shows the confirmation SMS. The server checks the
  // code against Safaricom's record (online), or flags it for reconciliation
  // (offline) — the till never trusts a typed code on its own.
  import { formatMoney } from '../money';
  import { type Tender, fromCents, toCents } from './tender.svelte';

  let {
    tender,
    currency,
    online,
    ontake,
  }: {
    tender: Tender;
    currency: string;
    online: boolean;
    ontake: (code: string, cents: number) => void;
  } = $props();

  let code = $state('');
  // svelte-ignore state_referenced_locally
  let amount = $state(String(Math.ceil(tender.remaining / 100)));

  const clean = $derived(code.trim().toUpperCase());
  const validCode = $derived(/^[A-Z0-9]{10}$/.test(clean));
  const cents = $derived(toCents(amount));
  const used = $derived(tender.hasReference(clean));
</script>

<section>
  <h3>Enter the code from the customer's SMS</h3>
  <div class="row">
    <label>
      <span>M-Pesa code</span>
      <input
        bind:value={code}
        maxlength="10"
        placeholder="e.g. SKL4H7Q2XZ"
        autocomplete="off"
        style:text-transform="uppercase"
      />
    </label>
    <label class="amt">
      <span>Amount on the SMS</span>
      <input bind:value={amount} inputmode="decimal" />
    </label>
  </div>
  {#if !online}
    <p class="warn">
      Offline: this code is checked when the till reconnects. If it does not match, the sale is flagged for
      the manager to reconcile.
    </p>
  {/if}
  {#if used}<p class="err">That code is already on this sale.</p>{/if}
  <button disabled={!validCode || cents <= 0 || used} onclick={() => ontake(clean, cents)}>
    Use code — {currency}
    {formatMoney(fromCents(cents))}
  </button>
</section>

<style>
  section {
    display: grid;
    gap: 12px;
  }
  h3 {
    margin: 0;
    font-size: 1.05rem;
  }
  .row {
    display: grid;
    grid-template-columns: 1.4fr 1fr;
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
    letter-spacing: 0.06em;
    color: var(--color-text);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
  }
  .warn {
    margin: 0;
    padding: 10px 12px;
    border-radius: var(--radius-sm);
    background: color-mix(in srgb, var(--color-warning) 14%, transparent);
    font-size: var(--text-sm);
  }
  .err {
    margin: 0;
    color: var(--color-danger);
    font-weight: 600;
  }
  button {
    height: 56px;
    font: inherit;
    font-size: 1.1rem;
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
