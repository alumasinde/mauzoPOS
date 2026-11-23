<script lang="ts">
  // Transaction lookup — find a sale by receipt number or id (this branch only).
  import { desktop } from '../desktop';
  import { formatMoney } from '../money';
  import ReprintButton from './ReprintButton.svelte';

  let { currency = 'KSh', onclose }: { currency?: string; onclose: () => void } = $props();

  const bridge = desktop();
  let ref = $state('');
  let busy = $state(false);
  let error = $state('');
  let sale = $state<{
    id: string;
    receipt_no: number;
    status: string;
    cashier: string;
    customer: string;
    total: string;
    paid: string;
    change_due: string;
    sold_at: string;
    items?: { name: string; qty: string; total: string }[];
    payments?: { method: string; amount: string; reference?: string }[];
  } | null>(null);

  async function find() {
    if (!bridge || !ref.trim() || busy) return;
    busy = true;
    error = '';
    sale = null;
    try {
      sale = (await bridge.SaleDetail(ref.trim())) as typeof sale;
      if (!sale) error = 'Sale not found on this terminal.';
    } catch {
      error = 'Sale not found — check the number, or the till may be offline.';
    } finally {
      busy = false;
    }
  }
</script>

<div class="modal">
  <h3>Find transaction</h3>
  <div class="pcRow">
    <input
      bind:value={ref}
      placeholder="Receipt no or sale id"
      inputmode="numeric"
      onkeydown={(e) => e.key === 'Enter' && find()}
    />
    <button class="go sm" onclick={find} disabled={busy || !ref.trim()}>{busy ? '…' : 'Find'}</button>
  </div>
  {#if error}<p class="err">{error}</p>{/if}
  {#if sale}
    <div class="found">
      <div class="line">
        <b>#{sale.receipt_no}</b>
        <span class="st {sale.status}">{sale.status.replace('_', ' ')}</span>
        <span class="amt">{currency} {formatMoney(sale.total)}</span>
      </div>
      <div class="meta">
        {new Date(sale.sold_at).toLocaleString()} · {sale.cashier || '—'}
        {#if sale.customer}
          · {sale.customer}{/if}
      </div>
      {#if sale.items?.length}
        <ul class="items">
          {#each sale.items as it, i (i)}
            <li><span>{it.qty} × {it.name}</span><span>{formatMoney(it.total)}</span></li>
          {/each}
        </ul>
      {/if}
      {#if sale.payments?.length}
        <div class="pays">
          {#each sale.payments as p, i (i)}
            <span class="pay">
              {p.method}{p.reference ? ` ${p.reference}` : ''} · {formatMoney(p.amount)}
            </span>
          {/each}
        </div>
      {/if}
      <div class="rp"><ReprintButton receipt={sale.receipt_no} /></div>
    </div>
  {/if}
  <button class="ghost" onclick={onclose}>Close</button>
</div>

<style>
  .rp {
    margin-top: var(--space-3);
  }
  .modal {
    width: min(440px, 92vw);
    background: var(--color-surface);
    border-radius: var(--radius-lg);
    padding: var(--space-5);
    box-shadow: 0 12px 40px rgb(0 0 0 / 0.3);
  }
  h3 {
    margin: 0 0 var(--space-3);
  }
  .pcRow {
    display: flex;
    gap: var(--space-2);
    margin-bottom: var(--space-3);
  }
  input {
    flex: 1;
    padding: 10px;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    font: inherit;
  }
  .go.sm {
    flex: none;
    padding: 8px 16px;
    border: 0;
    background: var(--color-primary);
    color: #fff;
    border-radius: var(--radius-md);
    cursor: pointer;
    font-weight: 600;
  }
  .err {
    color: var(--color-danger);
    font-size: var(--text-sm);
  }
  .found {
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    padding: var(--space-3);
    margin-bottom: var(--space-3);
  }
  .line {
    display: flex;
    align-items: center;
    gap: var(--space-3);
  }
  .st {
    font-size: var(--text-xs);
    font-weight: 600;
    text-transform: capitalize;
    padding: 2px 8px;
    border-radius: 999px;
    background: var(--color-surface-muted);
  }
  .st.completed {
    color: var(--color-success);
  }
  .st.voided {
    color: var(--color-danger);
  }
  .amt {
    margin-left: auto;
    font-weight: 700;
    font-variant-numeric: tabular-nums;
  }
  .meta {
    color: var(--color-text-muted);
    font-size: var(--text-xs);
    margin: var(--space-1) 0 var(--space-2);
  }
  .items {
    list-style: none;
    margin: 0;
    padding: var(--space-2) 0 0;
    border-top: 1px solid var(--color-border);
    font-size: var(--text-sm);
  }
  .items li {
    display: flex;
    justify-content: space-between;
    padding: 2px 0;
  }
  .pays {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-2);
    margin-top: var(--space-2);
    padding-top: var(--space-2);
    border-top: 1px solid var(--color-border);
  }
  .pay {
    font-size: var(--text-xs);
    background: var(--color-surface-muted);
    border-radius: 999px;
    padding: 2px 10px;
    text-transform: capitalize;
  }
  .ghost {
    width: 100%;
    padding: 10px;
    border: 1px solid var(--color-border);
    background: none;
    border-radius: var(--radius-md);
    cursor: pointer;
    font: inherit;
  }
</style>
