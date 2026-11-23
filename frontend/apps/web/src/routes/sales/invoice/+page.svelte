<script lang="ts">
  // A4 tax invoice / receipt for one sale — print it or "Save as PDF" from
  // the browser's print dialog. Data comes from the same receipt document the
  // till prints, so totals and VAT always agree with the thermal receipt.
  import { onMount } from 'svelte';
  import { page } from '$app/state';
  import { ApiError } from '@mauzo/api-client';
  import type { ReceiptDoc } from '@mauzo/types';
  import { Button, Icon } from '@mauzo/ui';
  import { api } from '#lib/api.ts';
  import InvoiceA4 from '#lib/receipts/InvoiceA4.svelte';

  let doc = $state<ReceiptDoc | null>(null);
  let status = $state('');
  let error = $state<string | null>(null);

  const id = $derived(page.url.searchParams.get('id') ?? '');

  onMount(async () => {
    if (!id) {
      error = 'No sale selected.';
      return;
    }
    try {
      const res = await api.get<{ doc: ReceiptDoc; status: string }>(
        `/api/v1/t/receipts/sales/${encodeURIComponent(id)}/invoice`,
      );
      doc = res.doc;
      status = res.status;
    } catch (e) {
      error = e instanceof ApiError ? e.message : 'Could not load the invoice';
    }
  });
</script>

<svelte:head>
  <title>{doc ? `Invoice ${doc.sale.receipt_no} — ${doc.business.name}` : 'Invoice'}</title>
</svelte:head>

<div class="bar noprint">
  <Button onclick={() => window.print()} disabled={!doc}>
    <Icon name="printer" size={16} /> Print / Save as PDF
  </Button>
  <span class="muted">Choose “Save as PDF” as the printer to get a PDF file.</span>
</div>

{#if error}
  <p class="err">{error}</p>
{:else if !doc}
  <p class="muted">Loading…</p>
{:else}
  <InvoiceA4 {doc} {status} />
{/if}

<style>
  .bar {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    margin-bottom: var(--space-4);
  }
  .muted {
    color: var(--color-text-muted);
  }
  .err {
    color: var(--color-danger);
  }
  @media print {
    @page {
      size: A4;
      margin: 0;
    }
    .noprint,
    :global(.side),
    :global(.topbar) {
      display: none !important;
    }
    :global(main) {
      padding: 0 !important;
      max-width: none !important;
    }
  }
</style>
