<script lang="ts">
  // Transaction drawer: full detail, the Daraja event timeline, and the
  // allocation/reversal actions gated by the caller's permissions.
  import { ApiError } from '@mauzo/api-client';
  import { resolve } from '$app/paths';
  import type { Sale } from '@mauzo/types';
  import { Badge, Button, Icon, Modal, toast } from '@mauzo/ui';
  import { formatMoney } from '@mauzo/utils';
  import { api } from '#lib/api.ts';
  import { mpesaApi, type MpesaEvent, type MpesaSaleBrief, type MpesaSuggestion, type MpesaTxn } from './api';

  let {
    txn = $bindable<MpesaTxn | null>(null),
    suggestions = [] as MpesaSuggestion[],
    canAllocate = false,
    canReallocate = false,
    onchange,
  }: {
    txn: MpesaTxn | null;
    suggestions: MpesaSuggestion[];
    canAllocate: boolean;
    canReallocate: boolean;
    onchange: () => void;
  } = $props();

  let events = $state<MpesaEvent[]>([]);
  let sale = $state<MpesaSaleBrief | null>(null);
  let receipt = $state('');
  let matches = $state<Sale[]>([]);
  let searching = $state(false);
  let busy = $state(false);
  let confirmReverse = $state(false);
  let reverseReason = $state('');

  const open = $derived(txn !== null);
  const mine = $derived(suggestions.filter((s) => s.txn.public_id === txn?.public_id));
  const allocatable = $derived(txn?.status === 'success' && !txn.allocated_sale_id);
  const reversible = $derived(txn?.status === 'success');

  $effect(() => {
    if (!txn) return;
    const id = txn.public_id;
    void mpesaApi.get(id).then((d) => {
      events = d.events;
      sale = d.sale;
    });
  });

  async function searchSales() {
    if (!receipt.trim()) return;
    searching = true;
    try {
      const res = await api.list<Sale>('/api/v1/t/sales', { search: receipt.trim(), per_page: 8 });
      matches = res.data.filter((s) => s.status !== 'voided');
    } catch {
      matches = [];
    } finally {
      searching = false;
    }
  }

  async function link(saleID: string, reallocate = false) {
    if (!txn || busy) return;
    busy = true;
    try {
      if (reallocate) await mpesaApi.reallocate(txn.public_id, saleID);
      else await mpesaApi.allocate(txn.public_id, saleID);
      toast.success(reallocate ? 'Re-allocated.' : 'Allocated to sale.');
      onchange();
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Allocation failed');
    } finally {
      busy = false;
    }
  }

  async function reverse() {
    if (!txn || busy) return;
    busy = true;
    try {
      await mpesaApi.reverse(txn.public_id, reverseReason);
      toast.success('Reversal requested — Daraja confirms asynchronously.');
      confirmReverse = false;
      onchange();
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Reversal failed');
    } finally {
      busy = false;
    }
  }

  function close() {
    txn = null;
    events = [];
    sale = null;
    receipt = '';
    matches = [];
    confirmReverse = false;
    reverseReason = '';
  }
</script>

<Modal {open} title="M-Pesa transaction" size="lg" onclose={close}>
  {#if txn}
    <div class="head">
      <div class="amt">KSh {formatMoney(txn.amount)}</div>
      <Badge>{txn.status.replace('_', ' ')}</Badge>
    </div>
    <dl class="grid">
      <div>
        <dt>Code</dt>
        <dd><code>{txn.trans_id ?? '—'}</code></dd>
      </div>
      <div>
        <dt>Phone</dt>
        <dd>{txn.phone || '—'}</dd>
      </div>
      <div>
        <dt>Payer</dt>
        <dd>{txn.payer_name || '—'}</dd>
      </div>
      <div>
        <dt>Type</dt>
        <dd>{txn.kind === 'stk' ? 'STK push' : 'Till/Paybill (C2B)'}</dd>
      </div>
      <div>
        <dt>Account ref</dt>
        <dd>{txn.account_ref || '—'}</dd>
      </div>
      <div>
        <dt>Shortcode</dt>
        <dd>{txn.short_code || '—'}</dd>
      </div>
      <div>
        <dt>Received</dt>
        <dd>{txn.trans_time ?? txn.created_at}</dd>
      </div>
      {#if txn.result_desc}<div>
          <dt>Result</dt>
          <dd>{txn.result_desc}</dd>
        </div>{/if}
    </dl>

    {#if sale}
      <div class="salebox">
        <Icon name="receipt" size={16} />
        Allocated to sale <a href={resolve('/sales')}>receipt #{sale.receipt_no}</a>
      </div>
    {/if}

    {#if allocatable && (canAllocate || canReallocate)}
      <h4>Allocate to a sale</h4>
      {#if mine.length > 0}
        <div class="sugg">
          {#each mine as s (s.candidate.public_id)}
            <div class="srow">
              <span>
                <b>Receipt #{s.candidate.receipt_no}</b> · {s.candidate.branch_name} ·
                {s.candidate.cashier_name || 'till'} · KSh {formatMoney(s.candidate.mpesa_amount)}
              </span>
              {#if s.strength === 'weak'}<Badge>amount only</Badge>{/if}
              <Button size="sm" onclick={() => link(s.candidate.public_id)} disabled={busy}>Allocate</Button>
            </div>
          {/each}
        </div>
      {/if}
      <div class="pick">
        <input
          bind:value={receipt}
          placeholder="Receipt # — search sales"
          onkeydown={(e) => e.key === 'Enter' && searchSales()}
        />
        <Button variant="secondary" onclick={searchSales} disabled={searching}>
          {searching ? '…' : 'Search'}
        </Button>
      </div>
      {#each matches as s (s.id)}
        <div class="srow">
          <span>
            <b>#{s.receipt_no}</b> · {s.branch} · KSh {formatMoney(s.total)} · {s.sold_at
              .slice(0, 16)
              .replace('T', ' ')}
          </span>
          <Button size="sm" variant="secondary" onclick={() => link(s.id)} disabled={busy}>Allocate</Button>
        </div>
      {/each}
    {/if}

    {#if reversible && canReallocate}
      {#if confirmReverse}
        <div class="rev">
          <input bind:value={reverseReason} placeholder="Reason for the reversal" />
          <Button variant="danger" onclick={reverse} disabled={busy}>
            {busy ? '…' : 'Request reversal'}
          </Button>
          <Button variant="ghost" onclick={() => (confirmReverse = false)}>Keep</Button>
        </div>
      {:else}
        <button class="revlink" onclick={() => (confirmReverse = true)}>
          <Icon name="undo" size={14} /> Reverse this payment
        </button>
      {/if}
    {/if}

    {#if events.length > 0}
      <h4>Timeline</h4>
      <ul class="tl">
        {#each events as e (e.created_at + e.kind)}
          <li>
            <span class="ekind">{e.kind}</span>
            <span class="edet">{e.detail}{e.ref ? ` · ${e.ref}` : ''}</span>
            <time>{e.created_at.slice(11, 19)}</time>
          </li>
        {/each}
      </ul>
    {/if}
  {/if}
</Modal>

<style>
  .head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: var(--space-4);
  }
  .amt {
    font-size: var(--text-2xl);
    font-weight: 800;
  }
  .grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--space-3) var(--space-4);
    margin: 0 0 var(--space-4);
  }
  dt {
    font-size: var(--text-xs);
    color: var(--color-text-muted);
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }
  dd {
    margin: 0;
    font-weight: 600;
  }
  .salebox {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    background: color-mix(in srgb, var(--color-success) 8%, transparent);
    border: 1px solid color-mix(in srgb, var(--color-success) 30%, transparent);
    border-radius: var(--radius-md);
    padding: var(--space-3);
    margin-bottom: var(--space-4);
  }
  h4 {
    margin: var(--space-4) 0 var(--space-2);
    font-size: var(--text-sm);
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--color-text-muted);
  }
  .sugg {
    margin-bottom: var(--space-2);
  }
  .srow {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-3);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    padding: var(--space-2) var(--space-3);
    margin-bottom: var(--space-2);
    font-size: var(--text-sm);
  }
  .pick {
    display: flex;
    gap: var(--space-2);
    margin-bottom: var(--space-2);
  }
  .pick input,
  .rev input {
    flex: 1;
    padding: 8px 10px;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    font: inherit;
  }
  .rev {
    display: flex;
    gap: var(--space-2);
    margin-top: var(--space-3);
  }
  .revlink {
    display: inline-flex;
    align-items: center;
    gap: var(--space-1);
    border: 0;
    background: none;
    color: var(--color-danger);
    cursor: pointer;
    font-size: var(--text-sm);
    padding: 0;
    margin-top: var(--space-2);
  }
  .tl {
    list-style: none;
    margin: 0;
    padding: 0;
  }
  .tl li {
    display: flex;
    gap: var(--space-3);
    padding: var(--space-2) 0;
    border-bottom: 1px solid var(--color-border);
    font-size: var(--text-sm);
    align-items: baseline;
  }
  .ekind {
    font-weight: 700;
    color: var(--color-primary);
    min-width: 110px;
  }
  .edet {
    flex: 1;
    color: var(--color-text);
  }
  .tl time {
    color: var(--color-text-muted);
    font-size: var(--text-xs);
  }
</style>
