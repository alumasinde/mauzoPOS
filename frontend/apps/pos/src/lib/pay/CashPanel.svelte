<script lang="ts">
  // Cash: one tap on the note the customer handed over, or type the amount.
  import { formatMoney } from '../money';
  import Keypad from './Keypad.svelte';
  import { type Tender, fromCents, quickCash, toCents } from './tender.svelte';

  let { tender, currency }: { tender: Tender; currency: string } = $props();

  let given = $state('');
  const givenCents = $derived(toCents(given));
  const notes = $derived(quickCash(tender.remaining));
  const changeIf = $derived(Math.max(0, givenCents - tender.remaining));

  function take(cents: number) {
    if (tender.take('cash', cents)) given = '';
  }

  /** Enter on the keyboard takes what was typed. */
  export function submit(): boolean {
    if (givenCents <= 0) return false;
    take(givenCents);
    return true;
  }
  /** Typed digits from the physical keyboard. */
  export function type(k: string) {
    if (k === 'Backspace') given = given.slice(0, -1);
    else if (/^[0-9.]$/.test(k) && !(k === '.' && given.includes('.'))) given += k;
  }
</script>

<div class="cash">
  <div class="quick">
    <button class="exact" onclick={() => take(tender.remaining)}>
      <small>Exact</small>
      {formatMoney(fromCents(tender.remaining))}
    </button>
    {#each notes.filter((n) => n !== tender.remaining) as n (n)}
      <button onclick={() => take(n)}>{formatMoney(fromCents(n)).replace('.00', '')}</button>
    {/each}
  </div>

  <div class="given">
    <span>Amount given</span>
    <strong class:empty={!given}>{currency} {given ? formatMoney(given) : '0.00'}</strong>
    {#if givenCents > 0}
      <em class:short={givenCents < tender.remaining}>
        {givenCents >= tender.remaining
          ? `Change ${currency} ${formatMoney(fromCents(changeIf))}`
          : `Part payment — ${currency} ${formatMoney(fromCents(tender.remaining - givenCents))} still due`}
      </em>
    {/if}
  </div>

  <Keypad bind:value={given} />

  <button class="take" disabled={givenCents <= 0} onclick={() => take(givenCents)}>
    {givenCents >= tender.remaining || givenCents === 0 ? 'Take cash' : 'Take as part payment'}
    <kbd>Enter</kbd>
  </button>
</div>

<style>
  .cash {
    display: grid;
    gap: 14px;
  }
  .quick {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 8px;
  }
  .quick button {
    height: 64px;
    font: inherit;
    font-size: 1.35rem;
    font-weight: 800;
    font-variant-numeric: tabular-nums;
    color: var(--color-text);
    background: color-mix(in srgb, var(--color-success) 12%, var(--color-surface));
    border: 1px solid color-mix(in srgb, var(--color-success) 40%, transparent);
    border-radius: var(--radius-md);
    cursor: pointer;
  }
  .quick .exact {
    display: flex;
    flex-direction: column;
    justify-content: center;
    line-height: 1.1;
    background: var(--color-success);
    color: #fff;
    border: 0;
  }
  .exact small {
    font-size: 0.75rem;
    font-weight: 600;
    opacity: 0.9;
  }
  .given {
    display: grid;
    gap: 2px;
    padding: 12px 16px;
    border-radius: var(--radius-md);
    background: var(--color-surface-muted);
  }
  .given span {
    font-size: var(--text-sm);
    color: var(--color-text-muted);
  }
  .given strong {
    font-size: 2.2rem;
    font-variant-numeric: tabular-nums;
    line-height: 1.15;
  }
  .given strong.empty {
    color: var(--color-text-muted);
  }
  .given em {
    font-style: normal;
    font-weight: 700;
    color: var(--color-success);
  }
  .given em.short {
    color: var(--color-warning);
  }
  .take {
    height: 60px;
    font: inherit;
    font-size: 1.2rem;
    font-weight: 800;
    color: #fff;
    background: var(--color-primary);
    border: 0;
    border-radius: var(--radius-md);
    cursor: pointer;
  }
  .take:disabled {
    opacity: 0.45;
    cursor: default;
  }
  kbd {
    margin-left: 8px;
    font-size: 0.75rem;
    opacity: 0.75;
  }
</style>
