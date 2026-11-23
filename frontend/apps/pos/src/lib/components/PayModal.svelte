<script lang="ts">
  // Tender/payment modal — records the sale through the desktop bridge and
  // reports completion to the parent.
  import { desktop } from '../desktop';
  import { cart } from '../cart.svelte';
  import { formatMoney } from '../money';

  let {
    currency = 'KSh',
    method = 'cash',
    onclose,
    onpaid,
  }: {
    currency?: string;
    method?: string;
    onclose: () => void;
    onpaid: () => void;
  } = $props();

  const bridge = desktop();
  let tendered = $state('');
  let busy = $state(false);
  let error = $state('');

  const tenderedNum = $derived(Number(tendered) || 0);
  const totalNum = $derived(Number(cart.totals.total) || 0);
  const change = $derived(Math.max(0, tenderedNum - totalNum));

  async function complete() {
    if (!bridge || busy) return;
    busy = true;
    error = '';
    try {
      await bridge.Pay(method, cart.payload(), JSON.stringify(cart.totals));
      onpaid();
    } catch (e) {
      error = e instanceof Error ? e.message : 'Payment failed';
    } finally {
      busy = false;
    }
  }
</script>

<div class="overlay" role="presentation" onclick={(e) => e.target === e.currentTarget && onclose()}>
  <div class="modal">
    <h3>Tender — {method.toUpperCase()}</h3>
    <div class="due">Due: <b>{currency} {formatMoney(cart.totals.total)}</b></div>
    {#if method === 'cash'}
      <input class="big" bind:value={tendered} inputmode="decimal" placeholder="Tendered" />
      <div class="chg">Change: <b>{currency} {formatMoney(change.toFixed(2))}</b></div>
    {:else if method === 'mpesa'}
      <p class="muted">Prompt the customer on their phone — M-Pesa STK push lands in Phase 9.</p>
    {:else}
      <p class="muted">Confirm the {method} payment on the terminal device.</p>
    {/if}
    {#if error}<p class="err">{error}</p>{/if}
    <div class="row2">
      <button class="ghost" onclick={onclose}>Back</button>
      <button class="go" onclick={complete} disabled={busy || (method === 'cash' && tenderedNum < totalNum)}>
        {busy ? '…' : 'Complete sale'}
      </button>
    </div>
  </div>
</div>

<style>
  .overlay {
    position: fixed;
    inset: 0;
    background: rgb(4 10 24 / 0.55);
    display: grid;
    place-items: center;
    z-index: 40;
  }
  .modal {
    width: min(420px, 92vw);
    background: var(--color-surface);
    border-radius: var(--radius-lg);
    padding: var(--space-5);
    box-shadow: 0 12px 40px rgb(0 0 0 / 0.3);
  }
  h3 {
    margin: 0 0 var(--space-3);
  }
  .muted {
    color: var(--color-text-muted);
  }
  .err {
    color: var(--color-danger);
  }
  input {
    width: 100%;
    padding: var(--space-3);
    font: inherit;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    box-sizing: border-box;
  }
  input.big {
    font-size: var(--text-2xl);
    text-align: center;
  }
  .due {
    font-size: var(--text-xl);
    margin-bottom: var(--space-3);
  }
  .chg {
    font-size: var(--text-lg);
    margin-top: var(--space-2);
    color: var(--color-success);
  }
  .row2 {
    display: flex;
    gap: var(--space-2);
    margin-top: var(--space-3);
  }
  .row2 .go,
  .row2 .ghost {
    flex: 1;
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
    padding: var(--space-2) var(--space-4);
    font: inherit;
    background: none;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    cursor: pointer;
  }
</style>
