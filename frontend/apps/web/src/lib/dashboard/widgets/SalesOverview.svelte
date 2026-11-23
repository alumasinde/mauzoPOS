<script lang="ts">
  import { onMount } from 'svelte';
  import { SvelteDate } from 'svelte/reactivity';
  import { ApiError } from '@mauzo/api-client';
  import { AreaChart, Icon } from '@mauzo/ui';
  import type { AreaPoint } from '@mauzo/ui';
  import { formatMoney } from '@mauzo/utils';
  import WidgetCard from '../WidgetCard.svelte';
  import type { WidgetProps } from '../types';
  import { fetchDaily, windowFor, type DailyPoint } from './salesApi';

  let { widget, period, branchId }: WidgetProps = $props();

  let points = $state<AreaPoint[]>([]);
  let loading = $state(true);
  let error = $state<string | null>(null);

  // Fill gaps so the chart shows quiet days as zero, not missing.
  function fill(days: DailyPoint[]): AreaPoint[] {
    const { from, to } = windowFor(period);
    const byDay = new Map(days.map((d) => [d.day, Number(d.total)]));
    const out: AreaPoint[] = [];
    const d = new SvelteDate(from + 'T00:00:00');
    const end = new SvelteDate(to + 'T00:00:00');
    while (d <= end) {
      const key = d.toISOString().slice(0, 10);
      out.push({ label: key.slice(5), value: byDay.get(key) ?? 0 });
      d.setDate(d.getDate() + 1);
    }
    return out;
  }

  async function load() {
    loading = true;
    error = null;
    try {
      points = fill(await fetchDaily(period, branchId));
    } catch (e) {
      error = e instanceof ApiError ? e.message : 'Could not load sales';
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
  {#if points.every((p) => p.value === 0)}
    <div class="none">
      <Icon name="chart" size={28} />
      <p>No sales in this period yet.</p>
    </div>
  {:else}
    <AreaChart {points} format={(v) => formatMoney(String(v))} label="Sales total per day" />
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
