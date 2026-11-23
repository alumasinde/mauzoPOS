<script lang="ts">
  import { onMount } from 'svelte';
  import { ApiError } from '@mauzo/api-client';
  import type { PosTerminal } from '@mauzo/types';
  import { Icon } from '@mauzo/ui';
  import { api } from '#lib/api.ts';
  import WidgetCard from '../WidgetCard.svelte';
  import type { WidgetProps } from '../types';

  let { widget }: WidgetProps = $props();

  let terminals = $state<PosTerminal[]>([]);
  let loading = $state(true);
  let error = $state<string | null>(null);

  const activated = $derived(terminals.filter((t) => t.status === 'active'));
  const online = $derived(activated.filter((t) => t.online));
  const attention = $derived(terminals.filter((t) => t.status !== 'active'));

  function label(t: PosTerminal): string {
    if (t.status === 'pending') return 'Not activated';
    if (t.status === 'disabled') return 'Disabled';
    return t.online ? 'Online' : 'Offline';
  }
  function tone(t: PosTerminal): string {
    if (t.status === 'pending') return 'pending';
    if (t.status === 'disabled') return 'disabled';
    return t.online ? 'active' : 'offline';
  }

  onMount(async () => {
    try {
      terminals = await api.get<PosTerminal[]>('/api/v1/t/terminals');
    } catch (e) {
      error = e instanceof ApiError ? e.message : 'Could not load terminals';
    } finally {
      loading = false;
    }
  });
</script>

<WidgetCard title={widget.title} {loading} {error}>
  {#if terminals.length === 0}
    <div class="none">
      <Icon name="monitor" size={28} />
      <p>No POS terminals registered yet.</p>
    </div>
  {:else}
    <div class="counts">
      <div class="count">
        <span class="circle ok"><Icon name="plug" size={22} strokeWidth={2.4} /></span>
        <span><strong>{online.length}</strong><small>Online now</small></span>
      </div>
      <div class="count">
        <span class="circle warn"><Icon name="clock" size={22} strokeWidth={2.2} /></span>
        <span><strong>{attention.length}</strong><small>Need attention</small></span>
      </div>
    </div>
    <ul>
      {#each terminals.slice(0, 4) as t (t.id)}
        <li>
          <span class="name">{t.name}</span>
          <span class="bar {tone(t)}"></span>
          <span class="st {tone(t)}">{label(t)}</span>
        </li>
      {/each}
    </ul>
  {/if}
</WidgetCard>

<style>
  .counts {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--space-3);
    padding-bottom: var(--space-3);
    margin-bottom: var(--space-2);
    border-bottom: 1px solid var(--color-border);
  }
  .count {
    display: flex;
    align-items: center;
    gap: var(--space-3);
  }
  .count strong {
    display: block;
    font-size: var(--text-2xl);
    line-height: 1;
  }
  .count small {
    color: var(--color-text-muted);
    font-size: var(--text-xs);
  }
  .circle {
    display: grid;
    place-items: center;
    width: 44px;
    height: 44px;
    border-radius: 50%;
    color: #fff;
    flex: none;
  }
  .circle.ok {
    background: var(--color-success);
  }
  .circle.warn {
    background: var(--color-neutral);
  }
  ul {
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
    gap: var(--space-2);
  }
  li {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 1fr auto;
    align-items: center;
    gap: var(--space-3);
    font-size: var(--text-sm);
  }
  .name {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .bar {
    height: 8px;
    border-radius: 999px;
    background: var(--color-surface-muted);
  }
  .bar.active {
    background: var(--color-success);
  }
  .bar.offline {
    background: var(--color-neutral);
  }
  .bar.pending {
    background: var(--color-warning);
  }
  .st {
    font-weight: 600;
    font-size: var(--text-xs);
  }
  .st.active {
    color: var(--color-success);
  }
  .st.offline {
    color: var(--color-text-muted);
  }
  .st.pending {
    color: var(--color-warning);
  }
  .st.disabled {
    color: var(--color-text-muted);
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
