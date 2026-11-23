<script lang="ts">
  // Pill-styled select for filter bars: native control, custom chevron.
  import type { Snippet } from 'svelte';
  import type { HTMLSelectAttributes } from 'svelte/elements';
  import Icon from './Icon.svelte';

  interface Props extends HTMLSelectAttributes {
    value?: string;
    children: Snippet;
  }
  let { value = $bindable(''), children, ...rest }: Props = $props();
</script>

<div class="sel">
  <select bind:value {...rest}>{@render children()}</select>
  <span class="chev" aria-hidden="true"><Icon name="chevron-down" size={15} /></span>
</div>

<style>
  .sel {
    position: relative;
    display: inline-flex;
    align-items: center;
  }
  select {
    appearance: none;
    padding: 0.5rem 2rem 0.5rem 0.75rem;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    background: var(--color-surface);
    color: var(--color-text);
    font: inherit;
    font-size: var(--text-sm);
    cursor: pointer;
    max-width: 100%;
  }
  select:hover {
    border-color: var(--color-text-muted);
  }
  .chev {
    position: absolute;
    right: 0.55rem;
    pointer-events: none;
    color: var(--color-text-muted);
    display: grid;
  }
</style>
