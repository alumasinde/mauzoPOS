<script lang="ts">
  // Shift entry forms: open (float), cash in/out, close (blind count).
  // Close resolves to the Z-report via onclosed.
  import { Icon } from '@mauzo/ui';
  import { formatMoney } from '../money';
  import { openShift, shiftCash, closeShift, shift } from '../shift.svelte';
  import type { ShiftReport } from '../desktop';

  let {
    mode,
    currency = 'KSh',
    onclose,
    onclosed,
  }: {
    mode: 'open' | 'cash' | 'close';
    currency?: string;
    onclose: () => void;
    onclosed?: (report: ShiftReport) => void;
  } = $props();

  let amount = $state('');
  let reason = $state('');
  let direction = $state<'in' | 'out'>('out');
  let note = $state('');
  let error = $state('');
  let busy = $state(false);

  const titles = { open: 'Open shift', cash: 'Cash in / out', close: 'Close shift' };

  async function submit() {
    error = '';
    if (Number(amount) < 0 || (mode !== 'cash' && amount === '')) {
      error = 'Enter an amount';
      return;
    }
    busy = true;
    try {
      if (mode === 'open') {
        await openShift(amount || '0');
        onclose();
      } else if (mode === 'cash') {
        await shiftCash(direction, amount, reason.trim());
        onclose();
      } else {
        const rep = await closeShift(amount, note.trim());
        onclosed?.(rep);
        onclose();
      }
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed';
    } finally {
      busy = false;
    }
  }
</script>

<div class="modal">
  <h3>{titles[mode]}</h3>
  {#if mode === 'open'}
    <p class="muted">Count the float in the drawer to start the shift.</p>
  {:else if mode === 'close'}
    <p class="muted">Count the cash in the drawer — don't look at the expected figure first.</p>
    {#if shift.current}
      <p class="muted">
        Shift by {shift.current.opened_by_name} · float {currency}
        {formatMoney(shift.current.float)}
      </p>
    {/if}
  {/if}
  {#if mode === 'cash'}
    <div class="dirRow">
      <button class="dir" class:on={direction === 'in'} onclick={() => (direction = 'in')}>
        <Icon name="arrow-down" size={16} /> Cash in
      </button>
      <button class="dir" class:on={direction === 'out'} onclick={() => (direction = 'out')}>
        <Icon name="arrow-up" size={16} /> Cash out
      </button>
    </div>
  {/if}
  <input
    class="big"
    bind:value={amount}
    placeholder={mode === 'open' ? 'Float amount' : mode === 'close' ? 'Counted cash' : 'Amount'}
    inputmode="decimal"
    onkeydown={(e) => e.key === 'Enter' && submit()}
  />
  {#if mode === 'cash'}
    <input bind:value={reason} placeholder="Reason (e.g. paid supplier, bank drop)" />
  {:else if mode === 'close'}
    <input bind:value={note} placeholder="Note (optional)" />
  {/if}
  {#if error}<p class="err">{error}</p>{/if}
  <div class="row2">
    <button class="ghost" onclick={onclose}>Cancel</button>
    <button class="go" onclick={submit} disabled={busy}>
      {#if busy}<Icon name="refresh" size={16} />{:else}
        {mode === 'open' ? 'Open shift' : mode === 'cash' ? 'Record' : 'Close & report'}
      {/if}
    </button>
  </div>
</div>

<style>
  .modal {
    width: min(420px, 92vw);
    background: var(--color-surface);
    border-radius: var(--radius-lg);
    padding: var(--space-5);
    box-shadow: 0 12px 40px rgb(0 0 0 / 0.3);
  }
  h3 {
    margin: 0 0 var(--space-2);
  }
  .muted {
    color: var(--color-text-2);
    font-size: var(--text-sm);
    margin: 0 0 var(--space-3);
  }
  input {
    width: 100%;
    padding: var(--space-3);
    font: inherit;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    box-sizing: border-box;
    margin-bottom: var(--space-2);
  }
  input.big {
    font-size: var(--text-2xl);
    text-align: center;
  }
  .dirRow {
    display: flex;
    gap: var(--space-2);
    margin-bottom: var(--space-2);
  }
  .dir {
    flex: 1;
    padding: var(--space-3);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    background: var(--color-surface-2, var(--color-surface));
    font: inherit;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: var(--space-2);
    cursor: pointer;
  }
  .dir.on {
    border-color: var(--color-accent, #2f6fed);
    background: var(--color-accent-soft, #e8f0ff);
    font-weight: 600;
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
  .go:disabled {
    opacity: 0.6;
  }
  .err {
    color: var(--color-danger, #c0392b);
    font-size: var(--text-sm);
    margin: 0 0 var(--space-2);
  }
</style>
