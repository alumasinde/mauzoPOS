<script lang="ts">
  // A sale's receipt as the customer got it (the issued snapshot), its copy
  // history, an audited printed copy, and the A4 invoice.
  import { onDestroy, tick } from 'svelte';
  import { resolve } from '$app/paths';
  import { ApiError } from '@mauzo/api-client';
  import type { ReceiptOutput, SaleReceipt } from '@mauzo/types';
  import { Button, Modal, toast } from '@mauzo/ui';
  import { formatDate } from '@mauzo/utils';
  import { api } from '#lib/api.ts';
  import { auth } from '#lib/auth.svelte.ts';
  import ReceiptPaper from './ReceiptPaper.svelte';
  import { loadLogoURL } from './logo.ts';

  interface Props {
    open: boolean;
    saleId: string | null;
  }
  let { open = $bindable(), saleId }: Props = $props();

  let data = $state<SaleReceipt | null>(null);
  let shown = $state<ReceiptOutput | null>(null);
  let logoUrl = $state<string | null>(null);
  let loading = $state(false);
  let printing = $state(false);
  let printTarget = $state<ReceiptOutput | null>(null);

  const canReprint = $derived(auth.can('pos.receipt.reprint'));

  $effect(() => {
    if (open && saleId) load(saleId);
  });

  onDestroy(() => {
    if (logoUrl) URL.revokeObjectURL(logoUrl);
  });

  async function load(id: string) {
    loading = true;
    try {
      data = await api.get<SaleReceipt>(`/api/v1/t/receipts/sales/${id}`);
      shown = data.output;
      if (data.has_logo && !logoUrl) logoUrl = await loadLogoURL();
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Could not load the receipt');
      open = false;
    } finally {
      loading = false;
    }
  }

  // Printing from the back office hands out another copy, so it is a
  // numbered, audited reprint — never the original again.
  async function printCopy() {
    if (!saleId) return;
    printing = true;
    try {
      const res = await api.post<SaleReceipt>(`/api/v1/t/receipts/sales/${saleId}/reprint`);
      data = res;
      printTarget = res.output;
      await tick();
      window.print();
      toast.success(`Copy ${res.issued.copy_no} recorded`);
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Could not print a copy');
    } finally {
      printing = false;
      printTarget = null;
    }
  }

  const copyLabel = (n: number) => (n === 0 ? 'Original' : `Reprint ${n}`);
</script>

<Modal bind:open title="Receipt" size="lg">
  {#if loading || !data || !shown}
    <p class="muted">Loading…</p>
  {:else}
    <div class="wrap">
      <div class="paperbox"><ReceiptPaper output={shown} {logoUrl} /></div>
      <aside>
        <h4>Issued copies</h4>
        <ul>
          {#each data.copies ?? [] as c (c.id)}
            <li>
              <strong>{copyLabel(c.copy_no)}</strong>
              <small>
                {formatDate(c.created_at)} · {c.channel === 'pos' ? 'till' : 'back office'}{c.issued_by
                  ? ` · ${c.issued_by}`
                  : ''}
              </small>
            </li>
          {/each}
        </ul>
        <p class="muted small">
          Layout: {data.issued.template_name
            ? `${data.issued.template_name} v${data.issued.template_version}`
            : `default (${data.issued.builtin_key})`}
        </p>
        <div class="acts">
          {#if canReprint}
            <Button onclick={printCopy} loading={printing}>Print a copy</Button>
          {/if}
          {#if saleId}
            <!-- eslint-disable svelte/no-navigation-without-resolve -- resolve() plus a query string -->
            <a
              class="link"
              href={`${resolve('/sales/invoice')}?id=${encodeURIComponent(saleId)}`}
              target="_blank"
              rel="noopener"
            >
              A4 invoice / PDF
            </a>
            <!-- eslint-enable svelte/no-navigation-without-resolve -->
          {/if}
        </div>
      </aside>
    </div>
  {/if}
</Modal>

{#if printTarget}
  <div class="print-area"><ReceiptPaper output={printTarget} {logoUrl} printable /></div>
{/if}

<style>
  .muted {
    color: var(--color-text-muted);
  }
  .small {
    font-size: var(--text-sm);
  }
  .wrap {
    display: grid;
    grid-template-columns: minmax(0, auto) minmax(180px, 1fr);
    gap: var(--space-5);
    align-items: start;
  }
  .paperbox {
    display: flex;
    justify-content: center;
    padding: var(--space-4);
    background: var(--color-surface-muted);
    border-radius: var(--radius-md);
    max-height: 70vh;
    overflow: auto;
  }
  h4 {
    margin: 0 0 var(--space-2);
  }
  ul {
    list-style: none;
    margin: 0 0 var(--space-3);
    padding: 0;
    display: grid;
    gap: var(--space-2);
  }
  li {
    display: flex;
    flex-direction: column;
  }
  li small {
    color: var(--color-text-muted);
  }
  .acts {
    display: grid;
    gap: var(--space-2);
  }
  .link {
    color: var(--color-primary);
    font-weight: 600;
    text-decoration: none;
  }
  .print-area {
    display: none;
  }
  @media print {
    :global(body *) {
      visibility: hidden;
    }
    .print-area,
    .print-area :global(*) {
      visibility: visible;
    }
    .print-area {
      display: block;
      position: absolute;
      left: 0;
      top: 0;
    }
  }
  @media (max-width: 720px) {
    .wrap {
      grid-template-columns: 1fr;
    }
  }
</style>
