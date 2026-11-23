<script lang="ts">
  import { ApiError } from '@mauzo/api-client';
  import type { SaleDetail } from '@mauzo/types';
  import { Button, Modal, toast } from '@mauzo/ui';
  import { formatMoney } from '@mauzo/utils';
  import { api } from '#lib/api.ts';

  interface Props {
    open: boolean;
    sale: SaleDetail | null;
    ondone: () => void;
  }
  let { open = $bindable(), sale, ondone }: Props = $props();

  let lines = $state<{ line_no: number; qty: string; restock: boolean; max: number }[]>([]);
  let refundMethod = $state('cash');
  let reason = $state('');
  let busy = $state(false);
  let error = $state<string | null>(null);

  $effect(() => {
    if (open && sale) {
      lines = sale.items
        .filter((i) => Number(i.qty) - Number(i.returned_qty) > 0)
        .map((i) => ({
          line_no: i.line_no,
          qty: '0',
          restock: true,
          max: Number(i.qty) - Number(i.returned_qty),
        }));
      refundMethod = 'cash';
      reason = '';
      error = null;
    }
  });

  const refund = $derived.by(() => {
    if (!sale) return '0';
    let sum = 0;
    for (const l of lines) {
      const it = sale.items.find((i) => i.line_no === l.line_no);
      if (!it) continue;
      const unit = Number(it.total) / Number(it.qty);
      sum += unit * (Number(l.qty) || 0);
    }
    return sum.toFixed(2);
  });

  async function submit() {
    if (!sale) return;
    const used = lines.filter((l) => Number(l.qty) > 0);
    if (!used.length) {
      error = 'Enter at least one quantity to return';
      return;
    }
    for (const l of used) {
      if (Number(l.qty) > l.max) {
        error = `Line ${l.line_no}: qty exceeds what was sold`;
        return;
      }
    }
    busy = true;
    error = null;
    try {
      await api.post(`/api/v1/t/sales/${sale.id}/returns`, {
        reason,
        refund_method: refundMethod,
        lines: used.map((l) => ({ line_no: l.line_no, qty: l.qty, restock: l.restock })),
      });
      toast.success(`Refund of ${formatMoney(refund)} recorded`);
      open = false;
      ondone();
    } catch (e) {
      error = e instanceof ApiError ? e.message : 'Return failed';
    } finally {
      busy = false;
    }
  }
</script>

<Modal bind:open title={`Return items — sale #${sale?.receipt_no ?? ''}`} locked={busy} size="lg">
  {#if sale}
    <table class="items">
      <thead><tr><th>Item</th><th>Sold</th><th>Return qty</th><th>Restock</th></tr></thead>
      <tbody>
        {#each sale.items.filter((i) => Number(i.qty) - Number(i.returned_qty) > 0) as it (it.line_no)}
          {@const l = lines.find((x) => x.line_no === it.line_no)}
          <tr>
            <td>{it.name}</td>
            <td>{it.qty} @ {formatMoney(it.total)}</td>
            <td>
              <input
                class="qty"
                type="number"
                min="0"
                max={l?.max}
                step="any"
                value={l?.qty ?? '0'}
                oninput={(e) => {
                  if (l) l.qty = (e.target as HTMLInputElement).value;
                }}
              />
            </td>
            <td>
              <input
                type="checkbox"
                checked={l?.restock ?? true}
                onchange={(e) => {
                  if (l) l.restock = (e.target as HTMLInputElement).checked;
                }}
              />
            </td>
          </tr>
        {/each}
      </tbody>
    </table>

    <div class="row">
      <label class="field">
        <span>Refund method</span>
        <select bind:value={refundMethod}>
          <option value="cash">Cash</option>
          <option value="card">Card</option>
          <option value="bank">Bank transfer</option>
          <option value="credit">Customer credit</option>
        </select>
      </label>
      <label class="field grow">
        <span>Reason</span>
        <input bind:value={reason} placeholder="e.g. Damaged, wrong item" />
      </label>
    </div>

    <div class="foot">
      <strong>Refund: {formatMoney(refund)}</strong>
      {#if error}<p class="err">{error}</p>{/if}
      <div class="modal-actions">
        <Button variant="secondary" onclick={() => (open = false)}>Cancel</Button>
        <Button disabled={busy} onclick={submit}>{busy ? 'Processing…' : 'Process refund'}</Button>
      </div>
    </div>
  {/if}
</Modal>

<style>
  .items {
    width: 100%;
    border-collapse: collapse;
    font-size: 13px;
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
  .qty {
    width: 80px;
    padding: 6px 8px;
    border: 1px solid var(--line, #e3e7ee);
    border-radius: 6px;
    font: inherit;
    text-align: right;
  }
  .row {
    display: flex;
    gap: 12px;
    margin-top: 14px;
  }
  .field {
    display: block;
  }
  .field.grow {
    flex: 1;
  }
  .field span {
    display: block;
    font-size: 12px;
    color: var(--ink-3, #8a93a6);
    margin-bottom: 4px;
  }
  .field select,
  .field input {
    width: 100%;
    padding: 9px 10px;
    border-radius: 8px;
    border: 1px solid var(--line, #e3e7ee);
    font: inherit;
  }
  .foot {
    margin-top: 14px;
  }
  .err {
    color: var(--danger, #d64545);
    font-size: 13px;
  }
  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    margin-top: 10px;
  }
</style>
