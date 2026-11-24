<script lang="ts">
  // Big touch keypad for amounts and phone numbers.
  let {
    value = $bindable(''),
    decimal = true,
    max = 12,
  }: { value: string; decimal?: boolean; max?: number } = $props();

  const KEYS = ['7', '8', '9', '4', '5', '6', '1', '2', '3'];

  function press(k: string) {
    if (k === 'back') value = value.slice(0, -1);
    else if (k === 'clear') value = '';
    else if (k === '.') {
      if (decimal && !value.includes('.')) value = (value || '0') + '.';
    } else if (value.length < max) {
      const dot = value.indexOf('.');
      if (dot >= 0 && value.length - dot > 2) return; // two decimals max
      value = value === '0' ? k : value + k;
    }
  }
</script>

<div class="pad" role="group" aria-label="Keypad">
  {#each KEYS as k (k)}
    <button type="button" onclick={() => press(k)}>{k}</button>
  {/each}
  <button type="button" onclick={() => press(decimal ? '.' : '00')}>{decimal ? '.' : '00'}</button>
  <button type="button" onclick={() => press('0')}>0</button>
  <button type="button" class="fn" onclick={() => press('back')} aria-label="Delete">⌫</button>
  <button type="button" class="fn wide" onclick={() => press('clear')}>Clear</button>
</div>

<style>
  .pad {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 8px;
  }
  button {
    height: 64px;
    font: inherit;
    font-size: 1.6rem;
    font-weight: 700;
    font-variant-numeric: tabular-nums;
    color: var(--color-text);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    cursor: pointer;
    touch-action: manipulation;
  }
  button:active {
    transform: scale(0.97);
    background: var(--color-surface-muted);
  }
  .fn {
    font-size: 1.2rem;
    color: var(--color-text-muted);
  }
  .wide {
    grid-column: 1 / -1;
    height: 48px;
  }
</style>
