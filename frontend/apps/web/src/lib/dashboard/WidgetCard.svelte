<script lang="ts">
  import type { Snippet } from 'svelte';
  import { Icon } from '@mauzo/ui';

  interface Props {
    title: string;
    /** "View all" target (already resolved). */
    href?: string;
    /** Header controls (right side). */
    actions?: Snippet;
    /** Placeholder state for a widget whose module is not built yet. */
    pending?: { icon: string; note: string };
    loading?: boolean;
    error?: string | null;
    children?: Snippet;
  }
  let { title, href, actions, pending, loading = false, error = null, children }: Props = $props();
</script>

<section class="card" aria-label={title}>
  <header>
    <h2>{title}</h2>
    {#if actions}{@render actions()}{/if}
    {#if href && !pending}
      <!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- callers pass resolved hrefs -->
      <a {href}>View all <Icon name="arrow-right" size={15} /></a>
    {/if}
  </header>
  <div class="body">
    {#if pending}
      <div class="state">
        <span class="badge"><Icon name={pending.icon} size={26} /></span>
        <p>{pending.note}</p>
      </div>
    {:else if loading}
      <div class="state"><p>Loading…</p></div>
    {:else if error}
      <div class="state"><p class="err">{error}</p></div>
    {:else if children}
      {@render children()}
    {/if}
  </div>
</section>

<style>
  .card {
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-sm);
    padding: var(--space-4) var(--space-5);
    display: flex;
    flex-direction: column;
    min-width: 0;
    height: 100%;
  }
  header {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    margin-bottom: var(--space-3);
    min-height: 36px;
  }
  h2 {
    margin: 0;
    font-size: var(--text-lg);
    margin-right: auto;
  }
  a {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    color: var(--color-primary);
    font-size: var(--text-sm);
    font-weight: 600;
    text-decoration: none;
  }
  a:hover {
    text-decoration: underline;
  }
  .body {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-height: 0;
  }
  .state {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--space-3);
    text-align: center;
    color: var(--color-text-muted);
    padding: var(--space-5) var(--space-3);
    min-height: 180px;
  }
  .state p {
    margin: 0;
    max-width: 32ch;
    font-size: var(--text-sm);
  }
  .badge {
    display: grid;
    place-items: center;
    width: 52px;
    height: 52px;
    border-radius: 50%;
    background: var(--color-surface-muted);
    color: var(--color-primary);
  }
  .err {
    color: var(--color-danger);
  }
</style>
