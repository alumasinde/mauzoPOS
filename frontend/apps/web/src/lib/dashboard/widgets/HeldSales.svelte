<script lang="ts">
  import { onMount } from 'svelte';
  import { resolve } from '$app/paths';
  import { ApiError } from '@mauzo/api-client';
  import type { HeldSale } from '@mauzo/types';
  import { Icon } from '@mauzo/ui';
  import { formatDate } from '@mauzo/utils';
  import { api } from '#lib/api.ts';
  import WidgetCard from '../WidgetCard.svelte';
  import type { WidgetProps } from '../types';

  let { widget }: WidgetProps = $props();

  let held = $state<HeldSale[]>([]);
  let loading = $state(true);
  let error = $state<string | null>(null);

  onMount(async () => {
    try {
      held = await api.get<HeldSale[]>('/api/v1/t/sales/held');
    } catch (e) {
      error = e instanceof ApiError ? e.message : 'Could not load held sales';
    } finally {
      loading = false;
    }
  });
</script>

<WidgetCard title={widget.title} {loading} {error}>
  {#if held.length === 0}
    <div class="none">
      <Icon name="pause" size={28} />
      <p>Nothing parked at the tills right now.</p>
    </div>
  {:else}
    <ul>
      {#each held.slice(0, 6) as h (h.id)}
        <li>
          <span class="lbl">{h.label || 'Held sale'}</span>
          <span class="who">{h.held_by || h.terminal}</span>
          <span class="when">{formatDate(h.held_at)}</span>
        </li>
      {/each}
    </ul>
    {#if held.length > 6}
      <a class="all" href={resolve('/sales/held')}>+{held.length - 6} more →</a>
    {/if}
  {/if}
</WidgetCard>

<style>
  ul {
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
  }
  li {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto auto;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-2) 0;
    border-bottom: 1px solid var(--color-border);
    font-size: var(--text-sm);
  }
  li:last-child {
    border-bottom: 0;
  }
  .lbl {
    font-weight: 600;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .who {
    color: var(--color-text-muted);
  }
  .when {
    color: var(--color-text-muted);
    font-size: var(--text-xs);
  }
  .all {
    display: inline-block;
    margin-top: var(--space-3);
    font-size: var(--text-sm);
    color: var(--color-primary);
    text-decoration: none;
  }
  .none {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--space-2);
    min-height: 160px;
    color: var(--color-text-muted);
  }
  .none p {
    margin: 0;
    font-size: var(--text-sm);
  }
</style>
