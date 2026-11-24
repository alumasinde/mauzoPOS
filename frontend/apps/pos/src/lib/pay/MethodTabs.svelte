<script lang="ts">
  // Payment method cards: icon, name and what it is for, F1–F4 shortcuts.
  import { Icon } from '@mauzo/ui';
  import { METHOD_INFO } from './methods.svelte';
  import type { Method } from './tender.svelte';

  let { methods, active = $bindable() }: { methods: Method[]; active: Method } = $props();
</script>

<div class="wrap">
  <h3>Select payment method</h3>
  <nav class="methods" aria-label="Payment method">
    {#each methods as m, i (m)}
      <button class="m-{m}" class:on={active === m} aria-pressed={active === m} onclick={() => (active = m)}>
        {#if active === m}<span class="tick"><Icon name="check" size={14} /></span>{/if}
        <span class="ic"><Icon name={METHOD_INFO[m].icon} size={30} /></span>
        <strong>{METHOD_INFO[m].label}</strong>
        <small>{METHOD_INFO[m].hint}</small>
        {#if i < 4}<kbd>F{i + 1}</kbd>{/if}
      </button>
    {/each}
  </nav>
</div>

<style>
  .wrap {
    display: grid;
    gap: 8px;
  }
  h3 {
    margin: 0;
    font-size: 1.05rem;
  }
  .methods {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(104px, 1fr));
    gap: 8px;
  }
  button {
    position: relative;
    display: grid;
    justify-items: center;
    gap: 2px;
    padding: 12px 6px 8px;
    font: inherit;
    color: var(--color-text);
    background: var(--color-surface);
    border: 2px solid var(--color-border);
    border-radius: var(--radius-lg);
    cursor: pointer;
  }
  button:hover {
    border-color: var(--color-primary);
  }
  .ic {
    color: var(--color-primary);
  }
  .m-cash .ic,
  .m-mpesa .ic {
    color: #1f8f4e;
  }
  strong {
    font-size: 1.1rem;
  }
  small {
    font-size: 0.78rem;
    color: var(--color-text-muted);
    text-align: center;
  }
  kbd {
    position: absolute;
    top: 6px;
    left: 8px;
    font-size: 0.68rem;
    opacity: 0.55;
  }
  .on {
    border-color: var(--color-primary);
    background: color-mix(in srgb, var(--color-primary) 9%, var(--color-surface));
    box-shadow: 0 0 0 2px color-mix(in srgb, var(--color-primary) 25%, transparent);
  }
  .tick {
    position: absolute;
    top: 6px;
    right: 6px;
    display: grid;
    place-items: center;
    width: 22px;
    height: 22px;
    color: #fff;
    background: var(--color-primary);
    border-radius: 50%;
  }
</style>
