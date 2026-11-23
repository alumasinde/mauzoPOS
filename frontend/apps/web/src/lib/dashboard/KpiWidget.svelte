<script lang="ts">
  import { onMount } from 'svelte';
  import KpiCard from './KpiCard.svelte';
  import { PERIODS } from './period';
  import type { KpiData, KpiEntry, WidgetProps } from './types';

  let { entry, ...props }: WidgetProps & { entry: KpiEntry } = $props();

  let data = $state<KpiData | null>(null);
  let failed = $state(false);
  const compare = $derived(PERIODS.find((p) => p.key === props.period)?.compare ?? '');

  onMount(async () => {
    if (!entry.load) return;
    try {
      data = await entry.load(props);
    } catch {
      failed = true;
    }
  });
</script>

<KpiCard
  title={props.widget.title}
  icon={entry.icon}
  tone={entry.tone}
  value={data?.value ?? null}
  delta={data?.delta ?? null}
  trend={data?.trend ?? []}
  {compare}
  pendingNote={failed ? 'Could not load' : entry.load ? 'Loading…' : entry.pendingNote}
/>
