<script lang="ts">
  import { ApiError } from '@mauzo/api-client';
  import type { SaleDetail } from '@mauzo/types';
  import { Badge, Button, Modal, toast } from '@mauzo/ui';
  import { formatDate, formatMoney } from '@mauzo/utils';
  import { api } from '#lib/api.ts';
  import { auth } from '#lib/auth.svelte.ts';
  import ReturnModal from './ReturnModal.svelte';

  interface Props {
    open: boolean;
    saleId: string | null;
    onchanged: () => void;
  }
  let { open = $bindable(), saleId, onchanged }: Props = $props();

  let sale = $state<SaleDetail | null>(null);
  let loading = $state(false);
  let error = $state<string | null>(null);
  let voidOpen = $state(false);
  let voidReason = $state('');
  let returnOpen = $state(false);
  let busy = $state(false);

  const canVoid = $derived(auth.can('sales.sale.void'));
  const canRefund = $derived(auth.can('sales.refund.create'));
  const voidable = $derived(sale?.status === 'completed');
  const returnable = $derived(sale?.status === 'completed' || sale?.status === 'partially_returned');

  $effect(() => {
    if (open && saleId) load(saleId);
    if (!open) sale = null;
  });

  async function load(id: string) {
    loading = true;
    error = null;
    try {
      sale = await api.get<SaleDetail>(`/api/v1/t/sales/${id}`);
    } catch (e) {
      error = e instanceof ApiError ? e.message : 'Could not load the sale';
    } finally {
      loading = false;
    }
  }

  async function voidSale() {
    if (!sale || !voidReason.trim()) return;
    busy = true;
    try {
      await api.post(`/api/v1/t/sales/${sale.id}/void`, { reason: voidReason.trim() });
      toast.success(`Sale #${sale.receipt_no} voided — stock restored`);
      voidOpen = false;
      voidReason = '';
      await load(sale.id);
      onchanged();
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Void failed');
    } finally {
      busy = false;
    }
  }

  const STATUS = {
    completed: 'success',
    voided: 'danger',
    returned: 'neutral',
    partially_returned: 'warning',
  } as const;
</script>

<Modal bind:open title={sale ? `Sale #${sale.receipt_no}` : 'Sale'} size="lg">
  {#if loading}
    <p class="dim">Loading…</p>
  {:else if error}
    <p class="err">{error}</p>
  {:else if sale}
    <div class="meta">
      <div><span>Status</span><Badge tone={STATUS[sale.status]}>{sale.status.replace('_', ' ')}</Badge></div>
      <div><span>Branch</span><strong>{sale.branch}</strong></div>
      <div><span>Terminal</span><strong>{sale.terminal || '—'}</strong></div>
      <div><span>Cashier</span><strong>{sale.cashier || '—'}</strong></div>
      <div><span>Customer</span><strong>{sale.customer || 'Walk-in'}</strong></div>
      <div><span>Sold at</span><strong>{formatDate(sale.sold_at)}</strong></div>
      {#if sale.voided_at}
        <div><span>Voided</span><strong>{formatDate(sale.voided_at)}</strong></div>
        <div><span>Reason</span><strong>{sale.void_reason}</strong></div>
      {/if}
    </div>

    <table class="items">
      <thead>
        <tr><th>Item</th><th>Qty</th><th>Price</th><th>Discount</th><th>Total</th><th>Returned</th></tr>
      </thead>
      <tbody>
        {#each sale.items as it (it.line_no)}
          <tr>
            <td>
              {it.name}{#if it.sku}
                <span class="dim">· {it.sku}</span>{/if}
              {#if it.promotion}<span class="promo">{it.promotion}</span>{/if}
              {#if it.price_override}<span class="dim">(override)</span>{/if}
            </td>
            <td>{it.qty}</td>
            <td>{formatMoney(it.unit_price)}</td>
            <td>{Number(it.discount) ? formatMoney(it.discount) : '—'}</td>
            <td>{formatMoney(it.total)}</td>
            <td>{Number(it.returned_qty) ? it.returned_qty : '—'}</td>
          </tr>
        {/each}
      </tbody>
    </table>

    <div class="totals">
      <div><span>Subtotal</span><strong>{formatMoney(sale.subtotal)}</strong></div>
      <div><span>Discount</span><strong>{formatMoney(sale.discount)}</strong></div>
      <div><span>VAT (incl.)</span><strong>{formatMoney(sale.tax)}</strong></div>
      <div class="grand"><span>Total</span><strong>{formatMoney(sale.total)}</strong></div>
      {#if Number(sale.change_due)}<div>
          <span>Change</span><strong>{formatMoney(sale.change_due)}</strong>
        </div>{/if}
    </div>

    <h4>Payments</h4>
    <table class="items">
      <tbody>
        {#each sale.payments as p, i (i)}
          <tr class={p.status !== 'confirmed' ? 'neg' : ''}>
            <td class="cap">{p.method}</td>
            <td>{formatMoney(p.amount)}</td>
            <td>{p.tendered ? `tendered ${formatMoney(p.tendered)}` : ''}</td>
            <td>{p.reference || ''}</td>
            <td><Badge tone={p.status === 'confirmed' ? 'success' : 'neutral'}>{p.status}</Badge></td>
          </tr>
        {/each}
      </tbody>
    </table>

    {#if sale.returns.length}
      <h4>Returns</h4>
      <table class="items">
        <tbody>
          {#each sale.returns as r (r.id)}
            <tr>
              <td>Return #{r.return_no}</td>
              <td>{formatMoney(r.refund_amount)}</td>
              <td class="cap">{r.refund_method}</td>
              <td>{r.reason}</td>
              <td>{formatDate(r.created_at)}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}

    <div class="actions">
      {#if canRefund && returnable}
        <Button variant="secondary" onclick={() => (returnOpen = true)}>Return items…</Button>
      {/if}
      {#if canVoid && voidable}
        <Button variant="danger" onclick={() => (voidOpen = true)}>Void sale…</Button>
      {/if}
    </div>
  {/if}
</Modal>

<Modal bind:open={voidOpen} title="Void sale" locked={busy}>
  <p>
    Void sale <strong>#{sale?.receipt_no}</strong> for {sale ? formatMoney(sale.total) : ''}? Payments are
    cancelled and stock is restored. This is audited and cannot be undone.
  </p>
  <label class="field">
    <span>Reason (required)</span>
    <input bind:value={voidReason} placeholder="e.g. Wrong items, duplicate" />
  </label>
  <div class="modal-actions">
    <Button variant="secondary" onclick={() => (voidOpen = false)}>Cancel</Button>
    <Button variant="danger" disabled={busy || !voidReason.trim()} onclick={voidSale}>
      {busy ? 'Voiding…' : 'Void sale'}
    </Button>
  </div>
</Modal>

<ReturnModal
  bind:open={returnOpen}
  {sale}
  ondone={() => {
    if (sale) load(sale.id);
    onchanged();
  }}
/>

<style>
  .meta {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 10px;
    margin-bottom: 16px;
  }
  .meta div {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .meta span,
  .totals span {
    font-size: 11px;
    color: var(--ink-3, #8a93a6);
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }
  .items {
    width: 100%;
    border-collapse: collapse;
    font-size: 13px;
    margin: 8px 0 14px;
  }
  .items th {
    text-align: left;
    font-size: 11px;
    color: var(--ink-3, #8a93a6);
    padding: 4px 6px;
    border-bottom: 1px solid var(--line, #e3e7ee);
  }
  .items td {
    padding: 7px 6px;
    border-bottom: 1px solid var(--line, #e3e7ee);
  }
  .items td:nth-child(n + 2) {
    text-align: right;
  }
  .neg td {
    color: var(--danger, #d64545);
  }
  .promo {
    display: inline-block;
    font-size: 10px;
    background: var(--warn-bg, #fff3d6);
    color: var(--warn, #a06a00);
    padding: 1px 6px;
    border-radius: 6px;
    margin-left: 6px;
  }
  .totals {
    display: flex;
    gap: 22px;
    justify-content: flex-end;
    margin: 6px 0 14px;
  }
  .totals div {
    text-align: right;
  }
  .totals strong {
    display: block;
  }
  .grand strong {
    font-size: 17px;
    color: var(--accent, #2f6df6);
  }
  h4 {
    font-size: 13px;
    margin: 14px 0 4px;
  }
  .actions {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    margin-top: 16px;
  }
  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    margin-top: 14px;
  }
  .dim {
    color: var(--ink-3, #8a93a6);
  }
  .err {
    color: var(--danger, #d64545);
  }
  .cap {
    text-transform: capitalize;
  }
  .field {
    display: block;
    margin-top: 10px;
  }
  .field span {
    display: block;
    font-size: 12px;
    color: var(--ink-3, #8a93a6);
    margin-bottom: 4px;
  }
  .field input {
    width: 100%;
    padding: 9px 10px;
    border-radius: 8px;
    border: 1px solid var(--line, #e3e7ee);
    font: inherit;
  }
</style>
