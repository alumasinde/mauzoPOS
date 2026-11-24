<script lang="ts">
  // X/Z shift report — totals grid + optional print. X for an open shift,
  // Z after close (includes the blind count and variance).
  import { Icon } from '@mauzo/ui';
  import { formatMoney } from '../money';
  import { printShiftReport } from '../shift.svelte';
  import type { ShiftReport } from '../desktop';

  let {
    report,
    currency = 'KSh',
    onclose,
  }: {
    report: ShiftReport;
    currency?: string;
    onclose: () => void;
  } = $props();

  let printing = $state(false);
  let printErr = $state('');

  const z = $derived(report.shift.status === 'closed');
  const varianceNum = $derived(Number(report.shift.variance ?? 0));

  async function print() {
    printing = true;
    printErr = '';
    try {
      await printShiftReport(report);
    } catch (e) {
      printErr = e instanceof Error ? e.message : 'Print failed';
    } finally {
      printing = false;
    }
  }
</script>

<div class="modal">
  <div class="head">
    <h3>{z ? 'Z report — shift closed' : 'X report'}</h3>
    <span class="chip">{report.shift.terminal}</span>
  </div>
  <p class="muted">
    {report.shift.opened_by_name} · opened {new Date(report.shift.opened_at).toLocaleTimeString()}
    {#if report.shift.closed_at}
      · closed {new Date(report.shift.closed_at).toLocaleTimeString()}
    {/if}
  </p>

  <div class="block">
    <div class="r total">
      <span>Sales × {report.sales_count}</span><b>{currency} {formatMoney(report.sales_total)}</b>
    </div>
    {#each report.methods as m (m.method)}
      <div class="r"><span class="sub">{m.method} × {m.count}</span><span>{formatMoney(m.total)}</span></div>
    {/each}
  </div>

  <div class="block">
    <div class="r"><span>Float</span><span>{formatMoney(report.shift.float)}</span></div>
    <div class="r"><span>Cash sales</span><span>{formatMoney(report.cash_sales)}</span></div>
    <div class="r"><span>Cash in</span><span>+{formatMoney(report.cash_in)}</span></div>
    <div class="r"><span>Cash out</span><span>−{formatMoney(report.cash_out)}</span></div>
    <div class="r"><span>Cash refunds</span><span>−{formatMoney(report.cash_refunded)}</span></div>
    <div class="r total"><span>Expected in drawer</span><b>{currency} {formatMoney(report.expected)}</b></div>
    {#if z}
      <div class="r"><span>Counted</span><span>{formatMoney(report.shift.declared ?? '0')}</span></div>
      <div class="r variance" class:neg={varianceNum < 0} class:pos={varianceNum > 0}>
        <span>Variance</span><b>{varianceNum >= 0 ? '+' : ''}{formatMoney(report.shift.variance ?? '0')}</b>
      </div>
    {/if}
  </div>

  {#if report.movements.length > 0}
    <details class="moves">
      <summary>Movements ({report.movements.length})</summary>
      {#each report.movements as mv (mv.id)}
        <div class="r">
          <span class="sub">
            {new Date(mv.created_at).toLocaleTimeString()} · {mv.kind}
            {mv.reason && mv.reason !== 'float' ? `· ${mv.reason}` : ''}
          </span>
          <span>{mv.kind === 'drawer_open' ? '—' : formatMoney(mv.amount)}</span>
        </div>
      {/each}
    </details>
  {/if}

  {#if printErr}<p class="err">{printErr}</p>{/if}
  <div class="row2">
    <button class="ghost" onclick={print} disabled={printing}>
      {#if printing}<Icon name="refresh" size={16} />{:else}<Icon name="printer" size={16} />{/if}
      Print
    </button>
    <button class="go" onclick={onclose}>Done</button>
  </div>
</div>

<style>
  .modal {
    width: min(440px, 92vw);
    max-height: 86vh;
    overflow-y: auto;
    background: var(--color-surface);
    border-radius: var(--radius-lg);
    padding: var(--space-5);
    box-shadow: 0 12px 40px rgb(0 0 0 / 0.3);
  }
  .head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-2);
  }
  h3 {
    margin: 0;
  }
  .chip {
    background: var(--color-surface-2, #eef);
    border-radius: var(--radius-sm);
    padding: 2px 8px;
    font-size: var(--text-xs);
  }
  .muted {
    color: var(--color-text-2);
    font-size: var(--text-sm);
    margin: var(--space-1) 0 var(--space-3);
  }
  .block {
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    padding: var(--space-2) var(--space-3);
    margin-bottom: var(--space-3);
  }
  .r {
    display: flex;
    justify-content: space-between;
    padding: 4px 0;
    font-size: var(--text-sm);
  }
  .r .sub {
    color: var(--color-text-2);
  }
  .r.total {
    border-top: 1px dashed var(--color-border);
    margin-top: 4px;
    padding-top: 8px;
    font-weight: 600;
  }
  .r.variance.pos b {
    color: var(--color-danger, #c0392b);
  }
  .r.variance.neg b {
    color: var(--color-danger, #c0392b);
  }
  .moves {
    margin-bottom: var(--space-3);
    font-size: var(--text-sm);
  }
  .moves summary {
    cursor: pointer;
    color: var(--color-text-2);
    margin-bottom: var(--space-1);
  }
  .row2 {
    display: flex;
    gap: var(--space-2);
  }
  .ghost,
  .go {
    flex: 1;
    padding: var(--space-3);
    border-radius: var(--radius-md);
    font: inherit;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: var(--space-2);
  }
  .ghost {
    background: transparent;
    border: 1px solid var(--color-border);
  }
  .go {
    background: var(--color-accent, #2f6fed);
    color: #fff;
    border: none;
    font-weight: 600;
  }
  .err {
    color: var(--color-danger, #c0392b);
    font-size: var(--text-sm);
  }
</style>
