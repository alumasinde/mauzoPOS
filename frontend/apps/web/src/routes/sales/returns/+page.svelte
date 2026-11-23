<script lang="ts">
  import { onMount } from 'svelte';
  import { ApiError } from '@mauzo/api-client';
  import type { SaleReturn } from '@mauzo/types';
  import { DataTable, Pager, toast } from '@mauzo/ui';
  import { formatDate, formatMoney } from '@mauzo/utils';
  import { api } from '#lib/api.ts';

  const PER_PAGE = 25;
  let rows = $state<SaleReturn[]>([]);
  let loading = $state(true);
  let pageNum = $state(1);
  let total = $state(0);

  const columns = [
    { key: 'return_no', label: 'Return', width: '90px' },
    { key: 'receipt', label: 'Sale', width: '100px' },
    { key: 'when', label: 'Date', width: '150px' },
    { key: 'method', label: 'Refund via', width: '120px' },
    { key: 'amount', label: 'Amount', width: '130px', align: 'right' as const },
    { key: 'cashier', label: 'Processed by' },
    { key: 'reason', label: 'Reason' },
  ];

  async function load() {
    loading = true;
    try {
      const res = await api.list<SaleReturn>('/api/v1/t/sales/returns', {
        page: pageNum,
        per_page: PER_PAGE,
      });
      rows = res.data;
      total = res.meta.total;
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Failed to load returns');
    } finally {
      loading = false;
    }
  }

  onMount(load);
</script>

<div class="head">
  <h1>Returns &amp; refunds</h1>
  <p class="sub">Refunds processed against sales. Create one from a sale's detail page.</p>
</div>

<DataTable {columns} {rows} rowKey={(r) => r.id} {loading} empty="No returns recorded yet.">
  {#snippet cell(r: SaleReturn, col)}
    {#if col.key === 'return_no'}<strong>#{r.return_no}</strong>
    {:else if col.key === 'receipt'}#{r.receipt_no}
    {:else if col.key === 'when'}{formatDate(r.created_at)}
    {:else if col.key === 'method'}<span class="cap">{r.refund_method}</span>
    {:else if col.key === 'amount'}<span class="num">{formatMoney(r.refund_amount)}</span>
    {:else if col.key === 'cashier'}{r.cashier || '—'}
    {:else if col.key === 'reason'}{r.reason || '—'}
    {/if}
  {/snippet}
</DataTable>
<Pager
  page={pageNum}
  perPage={PER_PAGE}
  {total}
  onpage={(p) => {
    pageNum = p;
    load();
  }}
/>

<style>
  h1 {
    font-size: 22px;
    font-weight: 700;
    margin: 0 0 4px;
  }
  .sub {
    color: var(--ink-3, #8a93a6);
    font-size: 13px;
    margin: 0 0 16px;
  }
  .num {
    text-align: right;
    font-variant-numeric: tabular-nums;
  }
  .cap {
    text-transform: capitalize;
  }
</style>
