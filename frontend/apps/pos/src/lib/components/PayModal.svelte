<script lang="ts">
  // Tender/payment modal — builds the payments array, picks a customer for
  // credit sales, then posts through the desktop bridge (online-first, queued
  // offline). Shows the receipt number + change on success.
  import { untrack } from 'svelte';
  import { desktop, type PayResult } from '../desktop';
  import { cart } from '../cart.svelte';
  import { formatMoney } from '../money';
  import CustomerPick from './CustomerPick.svelte';
  import PaidDone from './PaidDone.svelte';

  let {
    currency = 'KSh',
    method = 'cash',
    onclose,
    onpaid,
  }: {
    currency?: string;
    method?: string;
    onclose: () => void;
    onpaid: (r: PayResult) => void;
  } = $props();

  const bridge = desktop();
  const METHODS: { key: string; label: string; ref: boolean; tendered: boolean }[] = [
    { key: 'cash', label: 'Cash', ref: false, tendered: true },
    { key: 'card', label: 'Card', ref: true, tendered: false },
    { key: 'bank', label: 'Bank', ref: true, tendered: false },
    { key: 'credit', label: 'Credit', ref: false, tendered: false },
  ];

  interface PayRow {
    method: string;
    amount: string;
    tendered: string;
    reference: string;
  }

  let rows = $state<PayRow[]>(untrack(() => [{ method, amount: '', tendered: '', reference: '' }]));
  let busy = $state(false);
  let error = $state('');
  let done = $state<PayResult | null>(null);

  const totalNum = $derived(Number(cart.totals.total) || 0);
  const paidNum = $derived(rows.reduce((n, r) => n + (Number(r.amount) || 0), 0));
  const remaining = $derived(Math.max(0, totalNum - paidNum));
  const tenderedTotal = $derived(
    rows.reduce((n, r) => n + (r.method === 'cash' ? Number(r.tendered || r.amount) || 0 : 0), 0),
  );
  const cashApplied = $derived(
    rows.reduce((n, r) => n + (r.method === 'cash' ? Number(r.amount) || 0 : 0), 0),
  );
  const change = $derived(Math.max(0, tenderedTotal - cashApplied));

  const needsCustomer = $derived(rows.some((r) => r.method === 'credit'));

  // Fill remaining balance into empty amounts.
  $effect(() => {
    const fill = remaining > 0 ? remaining : 0;
    for (const r of rows) if (!r.amount) r.amount = fill ? fill.toFixed(2) : '';
  });

  function addSplit() {
    const next = METHODS.find((m) => !rows.some((r) => r.method === m.key));
    rows = [...rows, { method: next?.key ?? 'cash', amount: '', tendered: '', reference: '' }];
  }

  function dropRow(i: number) {
    rows = rows.filter((_, x) => x !== i);
  }

  const CASH = { key: 'cash', label: 'Cash', ref: false, tendered: true };
  function spec(m: string) {
    return METHODS.find((x) => x.key === m) ?? CASH;
  }

  const valid = $derived(
    rows.length > 0 &&
      Math.abs(paidNum - totalNum) < 0.005 &&
      rows.every((r) => Number(r.amount) > 0 && (!spec(r.method).ref || r.reference.trim())) &&
      (!needsCustomer || !!cart.customerID),
  );

  async function complete() {
    if (!bridge || busy || !valid) return;
    busy = true;
    error = '';
    const payments = rows.map((r) => ({
      method: r.method,
      amount: r.amount,
      ...(r.method === 'cash' && r.tendered ? { tendered: r.tendered } : {}),
      ...(r.reference.trim() ? { reference: r.reference.trim() } : {}),
    }));
    try {
      const res = await bridge.Pay(
        JSON.stringify(payments),
        cart.payload(),
        '0',
        cart.customerID,
        cart.grant,
        '',
      );
      done = res;
    } catch (e) {
      error = e instanceof Error ? e.message : 'Payment failed';
    } finally {
      busy = false;
    }
  }

  function finish() {
    if (done) onpaid(done);
  }
</script>

<div class="overlay" role="presentation" onclick={(e) => e.target === e.currentTarget && !done && onclose()}>
  <div class="modal">
    {#if done}
      <PaidDone {currency} result={done} ondone={finish} />
    {:else}
      <h3>Tender — {currency} {formatMoney(cart.totals.total)}</h3>

      {#each rows as r, i (i)}
        <div class="payrow">
          <select bind:value={r.method}>
            {#each METHODS as m (m.key)}<option value={m.key}>{m.label}</option>{/each}
          </select>
          <input class="amt" bind:value={r.amount} inputmode="decimal" placeholder="Amount" />
          {#if rows.length > 1}
            <button class="x" onclick={() => dropRow(i)} aria-label="Remove">✕</button>
          {/if}
          {#if spec(r.method).tendered}
            <input
              class="wide"
              bind:value={r.tendered}
              inputmode="decimal"
              placeholder="Tendered (blank = exact)"
            />
          {/if}
          {#if spec(r.method).ref}
            <input
              class="wide"
              bind:value={r.reference}
              placeholder="{r.method === 'card' ? 'Card slip / approval ref' : 'Transfer reference'} *"
            />
          {/if}
        </div>
      {/each}

      <div class="sumrow">
        <span>Paid <b>{formatMoney(paidNum.toFixed(2))}</b></span>
        {#if remaining > 0.004}
          <span class="rem">Remaining <b>{formatMoney(remaining.toFixed(2))}</b></span>
        {:else if change > 0}
          <span class="chg">Change <b>{formatMoney(change.toFixed(2))}</b></span>
        {:else}
          <span class="ok">Exact</span>
        {/if}
        {#if rows.length < METHODS.length}
          <button class="split" onclick={addSplit}>+ Split</button>
        {/if}
      </div>

      <CustomerPick />
      {#if needsCustomer && !cart.customerID}
        <p class="err">Pick a customer — credit sales need an account.</p>
      {/if}

      {#if error}<p class="err">{error}</p>{/if}
      <div class="row2">
        <button class="ghost" onclick={onclose}>Back</button>
        <button class="go" onclick={complete} disabled={busy || !valid}>
          {busy ? '…' : `Complete sale — ${currency} ${formatMoney(cart.totals.total)}`}
        </button>
      </div>
    {/if}
  </div>
</div>

<style>
  .overlay {
    position: fixed;
    inset: 0;
    background: rgb(6 12 24 / 0.55);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 50;
  }
  .modal {
    width: min(480px, 94vw);
    background: var(--color-surface);
    border-radius: var(--radius-lg);
    padding: var(--space-5);
    box-shadow: 0 12px 40px rgb(0 0 0 / 0.3);
  }
  h3 {
    margin: 0 0 var(--space-3);
    font-size: var(--text-lg);
  }
  .payrow {
    display: grid;
    grid-template-columns: 110px 1fr auto;
    gap: var(--space-2);
    margin-bottom: var(--space-2);
  }
  .payrow .wide {
    grid-column: 1 / -1;
  }
  select,
  input {
    padding: 10px;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    font: inherit;
    font-size: var(--text-base);
    background: var(--color-surface);
  }
  .amt {
    font-variant-numeric: tabular-nums;
    text-align: right;
  }
  .x {
    border: 0;
    background: none;
    color: var(--color-danger);
    cursor: pointer;
    font-size: 14px;
  }
  .sumrow {
    display: flex;
    align-items: center;
    gap: var(--space-4);
    margin: var(--space-3) 0;
    font-size: var(--text-sm);
  }
  .rem {
    color: var(--color-warning);
  }
  .chg {
    color: var(--color-primary);
    font-weight: 600;
  }
  .ok {
    color: var(--color-success);
  }
  .split {
    margin-left: auto;
    border: 1px dashed var(--color-border);
    background: none;
    border-radius: var(--radius-md);
    padding: 6px 12px;
    font-size: var(--text-sm);
    cursor: pointer;
    color: var(--color-primary);
  }
  .err {
    color: var(--color-danger);
    font-size: var(--text-sm);
    margin: var(--space-2) 0;
  }
  .row2 {
    display: flex;
    gap: var(--space-2);
    margin-top: var(--space-3);
  }
  .ghost {
    flex: 1;
    padding: 12px;
    border: 1px solid var(--color-border);
    background: none;
    border-radius: var(--radius-md);
    cursor: pointer;
    font: inherit;
  }
  .go {
    flex: 2;
    padding: 12px;
    border: 0;
    background: var(--color-success);
    color: #fff;
    font-weight: 700;
    border-radius: var(--radius-md);
    cursor: pointer;
    font-size: var(--text-base);
  }
  .go:disabled {
    opacity: 0.5;
    cursor: default;
  }
</style>
