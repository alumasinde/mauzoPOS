<script lang="ts">
  import { onMount } from 'svelte';
  import { ApiError } from '@mauzo/api-client';
  import { DonutChart, Icon } from '@mauzo/ui';
  import { formatMoney } from '@mauzo/utils';
  import WidgetCard from '../WidgetCard.svelte';
  import type { WidgetProps } from '../types';
  import { fetchMethods, type MethodTotal } from './salesApi';

  let { widget, period, branchId }: WidgetProps = $props();

  let methods = $state<MethodTotal[]>([]);
  let loading = $state(true);
  let error = $state<string | null>(null);

  const TONES = ['#2f6fed', '#22a06b', '#8b5cf6', '#f59e0b', '#ef4444', '#14b8a6'];
  const slices = $derived(
    methods.map((m, i) => ({
      label: m.label ?? m.method,
      value: Number(m.total),
      color: TONES[i % TONES.length],
    })),
  );
  const grand = $derived(methods.reduce((n, m) => n + Number(m.total), 0));

  async function load() {
    loading = true;
    error = null;
    try {
      methods = await fetchMethods(period, branchId);
    } catch (e) {
      error = e instanceof ApiError ? e.message : 'Could not load payments';
    } finally {
      loading = false;
    }
  }

  onMount(load);
  $effect(() => {
    void period;
    void branchId;
    void load();
  });
</script>

<WidgetCard title={widget.title} {loading} {error}>
  {#if methods.length === 0}
    <div class="none">
      <Icon name="wallet" size={28} />
      <p>No payments recorded in this period.</p>
    </div>
  {:else}
    <DonutChart
      {slices}
      centerValue={formatMoney(String(grand))}
      centerLabel="collected"
      label="Payment methods"
    />
  {/if}
</WidgetCard>

<style>
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
