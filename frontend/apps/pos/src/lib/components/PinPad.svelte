<script lang="ts">
  // Reusable PIN entry: 4–6 digits, Enter/OK submits. Used for cashier
  // sign-in, unlock and supervisor authorisation.
  let {
    title = 'Enter PIN',
    subtitle = '',
    error = '',
    busy = false,
    onsubmit,
    oncancel,
  }: {
    title?: string;
    subtitle?: string;
    error?: string;
    busy?: boolean;
    onsubmit: (pin: string) => void;
    oncancel?: () => void;
  } = $props();

  let pin = $state('');

  function press(k: string) {
    if (busy) return;
    if (k === 'back') pin = pin.slice(0, -1);
    else if (k === 'clear') pin = '';
    else if (pin.length < 6) pin += k;
  }

  function submit() {
    if (pin.length >= 4 && !busy) onsubmit(pin);
  }

  $effect(() => {
    // Reset whenever a new error arrives or the pad remounts.
    if (error) pin = '';
  });
</script>

<div class="pad">
  <h2>{title}</h2>
  {#if subtitle}<p class="sub">{subtitle}</p>{/if}
  <div class="dots" aria-label="PIN">
    {#each [0, 1, 2, 3, 4, 5] as i (i)}
      <span class:filled={i < pin.length}></span>
    {/each}
  </div>
  {#if error}<p class="err">{error}</p>{/if}
  <div class="keys">
    {#each ['1', '2', '3', '4', '5', '6', '7', '8', '9', 'clear', '0', 'back'] as k (k)}
      <button
        class="key"
        class:fn={k === 'clear' || k === 'back'}
        disabled={busy}
        onclick={() => press(k)}
        aria-label={k === 'back' ? 'Backspace' : k === 'clear' ? 'Clear' : k}
      >
        {#if k === 'back'}⌫{:else if k === 'clear'}C{:else}{k}{/if}
      </button>
    {/each}
  </div>
  <div class="row">
    {#if oncancel}<button class="ghost" onclick={oncancel} disabled={busy}>Cancel</button>{/if}
    <button class="ok" onclick={submit} disabled={busy || pin.length < 4}>
      {busy ? 'Checking…' : 'OK'}
    </button>
  </div>
</div>

<style>
  .pad {
    width: 320px;
    margin: 0 auto;
    text-align: center;
  }
  h2 {
    margin: 0 0 var(--space-2);
    font-size: var(--text-xl);
  }
  .sub {
    color: var(--color-text-muted);
    margin: 0 0 var(--space-3);
  }
  .dots {
    display: flex;
    justify-content: center;
    gap: var(--space-2);
    margin: var(--space-4) 0;
  }
  .dots span {
    width: 14px;
    height: 14px;
    border-radius: 50%;
    border: 2px solid var(--color-border);
    background: transparent;
  }
  .dots span.filled {
    background: var(--color-primary);
    border-color: var(--color-primary);
  }
  .err {
    color: var(--color-danger);
    margin: var(--space-2) 0;
    min-height: 1.2em;
  }
  .keys {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: var(--space-2);
  }
  .key {
    padding: var(--space-4) 0;
    font-size: var(--text-xl);
    font-weight: 600;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    background: var(--color-surface);
    cursor: pointer;
  }
  .key:hover {
    background: var(--color-surface-muted);
  }
  .key.fn {
    font-size: var(--text-lg);
    color: var(--color-text-muted);
  }
  .row {
    display: flex;
    gap: var(--space-2);
    margin-top: var(--space-4);
  }
  .ok {
    flex: 1;
    padding: var(--space-3);
    font-size: var(--text-lg);
    font-weight: 700;
    background: var(--color-success);
    color: #fff;
    border: none;
    border-radius: var(--radius-md);
    cursor: pointer;
  }
  .ok:disabled {
    opacity: 0.5;
    cursor: default;
  }
  .ghost {
    padding: var(--space-3) var(--space-4);
    font-size: var(--text-lg);
    background: none;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    cursor: pointer;
  }
</style>
