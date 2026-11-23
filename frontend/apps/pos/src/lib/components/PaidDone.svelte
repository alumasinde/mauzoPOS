<script lang="ts">
  // Post-tender confirmation — receipt number + change, or the offline
  // queued state, plus the receipt printing outcome.
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
  let printed = $state(result.printed);
  let printError = $state(result.print_error ?? '');
  let busy = $state(false);

  // Print the original once (auto-print off, or the printer was down).
  async function printNow() {
    if (!bridge || busy) return;
    busy = true;
    printError = '';
    try {
      await bridge.PrintLast();
      printed = true;
    } catch (e) {
      printError = e instanceof Error ? e.message : 'Printing failed';
    } finally {
      busy = false;
    }
  }
</script>

<h3>{result.offline ? 'Sale queued' : 'Sale complete'}</h3>
<div class="donebox">
  {#if result.offline}
    <p class="muted">Offline — the sale will upload on the next sync. Reference:</p>
    <code class="key">{result.sale_id}</code>
  {:else}
    <div class="rcpt">Receipt <b>#{result.receipt_no}</b></div>
    <div class="tot">Total <b>{currency} {formatMoney(result.total)}</b></div>
  {/if}
  {#if Number(result.change_due) > 0}
    <div class="chg">Change due <b>{currency} {formatMoney(result.change_due)}</b></div>
  {/if}
</div>

<div class="print">
  {#if printed}
    <p class="ok">✓ Receipt printed{result.offline ? ' (provisional)' : ''}</p>
  {:else if result.can_print}
    {#if printError}<p class="err">Receipt not printed: {printError}</p>{/if}
    <button class="ghost" onclick={printNow} disabled={busy}>
      {busy ? 'Printing…' : printError ? 'Try printing again' : 'Print receipt'}
    </button>
  {:else if printError}
    <p class="err">Receipt not printed: {printError}</p>
  {/if}
  {#if printed && !result.offline && result.receipt_no}
    <ReprintButton receipt={result.receipt_no} label="Print another copy" />
  {/if}
</div>

<button class="go" onclick={ondone}>Done — new sale</button>

<style>
  h3 {
    margin: 0 0 var(--space-3);
    font-size: var(--text-lg);
  }
  .donebox {
    text-align: center;
    padding: var(--space-4) 0;
  }
  .muted {
    color: var(--color-text-muted);
    font-size: var(--text-sm);
  }
  .key {
    display: block;
    margin-top: var(--space-2);
    font-size: var(--text-xs);
    word-break: break-all;
  }
  .rcpt {
    font-size: var(--text-xl);
    margin-bottom: var(--space-2);
  }
  .tot {
    font-size: var(--text-lg);
  }
  .chg {
    margin-top: var(--space-2);
    color: var(--color-primary);
    font-size: var(--text-lg);
  }
  .print {
    display: grid;
    gap: var(--space-2);
    margin-bottom: var(--space-3);
  }
  .ok {
    margin: 0;
    text-align: center;
    color: var(--color-success);
    font-weight: 600;
  }
  .err {
    margin: 0;
    text-align: center;
    color: var(--color-danger);
    font-size: var(--text-sm);
  }
  .ghost {
    padding: 10px;
    font: inherit;
    font-weight: 600;
    background: none;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    cursor: pointer;
    color: var(--color-text);
  }
  .go {
    width: 100%;
    padding: 12px;
    border: 0;
    background: var(--color-success);
    color: #fff;
    font-weight: 700;
    border-radius: var(--radius-md);
    cursor: pointer;
    font-size: var(--text-base);
  }
</style>
