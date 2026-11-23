<script lang="ts">
  // Page header: tinted icon tile + title + subtitle, with an actions slot on
  // the right. Used at the top of every back-office page.
  import type { Snippet } from 'svelte';
  import Icon from './Icon.svelte';

  interface Props {
    title: string;
    sub?: string;
    icon?: string;
    actions?: Snippet;
  }
  let { title, sub = '', icon = 'box', actions }: Props = $props();
</script>

<div class="ph">
  <span class="tile" aria-hidden="true"><Icon name={icon} size={24} strokeWidth={1.7} /></span>
  <div class="txt">
    <h1>{title}</h1>
    {#if sub}<p class="sub">{sub}</p>{/if}
  </div>
  {#if actions}<div class="acts">{@render actions()}</div>{/if}
</div>

<style>
  .ph {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    margin-bottom: var(--space-5);
  }
  .tile {
    display: grid;
    place-items: center;
    width: 46px;
    height: 46px;
    border-radius: var(--radius-lg);
    background: color-mix(in srgb, var(--color-primary) 16%, var(--color-surface));
    color: var(--color-primary);
    border: 1px solid color-mix(in srgb, var(--color-primary) 30%, transparent);
    flex: none;
  }
  .txt {
    min-width: 0;
    flex: 1;
  }
  h1 {
    margin: 0;
    font-size: var(--text-xl);
    font-weight: 800;
    letter-spacing: -0.01em;
  }
  .sub {
    margin: 2px 0 0;
    font-size: var(--text-sm);
    color: var(--color-text-muted);
  }
  .acts {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    flex: none;
    flex-wrap: wrap;
    justify-content: flex-end;
  }
</style>
