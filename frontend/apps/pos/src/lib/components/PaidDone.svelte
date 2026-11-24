<script lang="ts">
  // Shown in the empty cart right after a sale: the change to hand over (big),
  // the receipt number, and what happened to the receipt (printed, saved as
  // PDF, or a button to print it). It goes away when the next item is scanned.
  import { onMount } from 'svelte';
  import { Icon } from '@mauzo/ui';
  import { desktop, type PayResult } from '../desktop';
  import { formatMoney } from '../money';
  import ReprintButton from './ReprintButton.svelte';

  let {
    currency = 'KSh',
    result,
    ondone,
  }: {
    currency?: string;
    result: PayResult;
    ondone: () => void;
  } = $props();

  const bridge = desktop();
  let printed = $state(false);
  let file = $state('');
  let printError = $state('');
  let busy = $state(false);

  onMount(() => {
    printed = result.printed;
    file = result.receipt_file ?? '';
    printError = result.print_error ?? '';
  });

  // Print the original once (auto-print off, or the printer was down).
  async function printNow() {
    if (!bridge || busy) return;
    busy = true;
    printError = '';
    try {
      file = (await bridge.PrintLast()) ?? '';
      printed = true;
    } catch (e) {
      printError = e instanceof Error ? e.message : 'Printing failed';
    } finally {
      busy = false;
    }
  }

  async function openFile() {
    try {
      await bridge?.OpenReceiptFile(file);
    } catch (e) {
      printError = e instanceof Error ? e.message : 'Could not open the receipt';
    }
  }

  const change = $derived(Number(result.change_due) > 0);
</script>

<div class="done">
  <div class="head">
    <span class="ck"><Icon name="check" size={22} /></span>
    <div>
      <strong>{result.offline ? 'Sale saved offline' : 'Sale complete'}</strong>
      <small>
        {result.offline ? 'Uploads on the next sync' : `Receipt #${result.receipt_no}`}
        · {currency}
        {formatMoney(result.total)}
      </small>
    </div>
    <button class="x" onclick={ondone} aria-label="Dismiss"><Icon name="x" size={18} /></button>
  </div>

  <div class="chg" class:none={!change}>
    <span>{change ? 'Give change' : 'No change due'}</span>
    <b>{currency} {formatMoney(result.change_due)}</b>
  </div>

  <div class="print">
    {#if printed && file}
      <p class="ok"><Icon name="check" size={15} /> Receipt saved as PDF</p>
      <button class="ghost" onclick={openFile}><Icon name="file" size={16} /> Open receipt</button>
    {:else if printed}
      <p class="ok">
        <Icon name="check" size={15} /> Receipt printed{result.offline ? ' (provisional)' : ''}
      </p>
    {:else if result.can_print}
      <button class="ghost" onclick={printNow} disabled={busy}>
        <Icon name="receipt" size={16} />
        {busy ? 'Printing…' : printError ? 'Try printing again' : 'Print receipt'}
      </button>
    {/if}
    {#if printError}<p class="err">Receipt not printed: {printError}</p>{/if}
    {#if printed && !result.offline && result.receipt_no}
      <ReprintButton receipt={result.receipt_no} label="Print another copy" />
    {/if}
  </div>

  <p class="next">Scan the next item to start a new sale.</p>
</div>

<style>
  .done {
    display: grid;
    gap: 12px;
    margin: 12px;
    padding: 14px;
    background: var(--color-surface);
    border: 1px solid color-mix(in srgb, var(--color-success) 40%, transparent);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-md, 0 4px 14px rgb(0 0 0 / 0.08));
  }
  .head {
    display: flex;
    align-items: center;
    gap: 10px;
  }
  .head div {
    flex: 1;
    display: grid;
  }
  .head strong {
    font-size: 1.15rem;
  }
  .head small {
    color: var(--color-text-muted);
  }
  .ck {
    display: grid;
    place-items: center;
    width: 38px;
    height: 38px;
    color: #fff;
    background: var(--color-success);
    border-radius: 50%;
  }
  .x {
    display: grid;
    place-items: center;
    width: 34px;
    height: 34px;
    color: var(--color-text-muted);
    background: var(--color-surface-muted);
    border: 0;
    border-radius: 50%;
    cursor: pointer;
  }
  .chg {
    display: grid;
    padding: 12px 16px;
    color: #fff;
    background: var(--color-success);
    border-radius: var(--radius-md);
  }
  .chg span {
    font-weight: 700;
    opacity: 0.9;
  }
  .chg b {
    font-size: 2.4rem;
    line-height: 1.1;
    font-variant-numeric: tabular-nums;
  }
  .chg.none {
    color: var(--color-success);
    background: color-mix(in srgb, var(--color-success) 10%, var(--color-surface));
  }
  .chg.none b {
    font-size: 1.5rem;
  }
  .print {
    display: grid;
    gap: 8px;
  }
  .ok {
    display: flex;
    gap: 6px;
    align-items: center;
    margin: 0;
    color: var(--color-success);
    font-weight: 600;
  }
  .err {
    margin: 0;
    color: var(--color-danger);
    font-size: var(--text-sm);
  }
  .ghost {
    display: flex;
    gap: 8px;
    align-items: center;
    justify-content: center;
    padding: 10px;
    font: inherit;
    font-weight: 700;
    color: var(--color-text);
    background: none;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    cursor: pointer;
  }
  .next {
    margin: 0;
    text-align: center;
    color: var(--color-text-muted);
    font-size: var(--text-sm);
  }
</style>
