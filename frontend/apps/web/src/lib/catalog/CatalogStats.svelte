<script lang="ts">
  // KPI row for the products page — every number comes from /catalog/overview.
  import type { CatalogOverview } from '@mauzo/types';
  import { StatCard } from '@mauzo/ui';

  let { stats }: { stats: CatalogOverview } = $props();
</script>

<div class="stats">
  <StatCard
    label="Total products"
    value={String(stats.total)}
    icon="box"
    tone="blue"
    delta={stats.delta_pct}
    deltaLabel="vs 30 days ago"
    trend={stats.trend_total}
  />
  <StatCard
    label="Active"
    value={String(stats.active)}
    icon="check-circle"
    tone="green"
    delta={stats.delta_active}
    deltaLabel="vs 30 days ago"
    trend={stats.trend_active}
  />
  <StatCard
    label="Low stock"
    value={String(stats.low_stock)}
    icon="alert"
    tone="orange"
    trend={stats.trend_low_stock}
  />
  <StatCard
    label="Archived"
    value={String(stats.archived)}
    icon="inbox"
    tone="purple"
    trend={stats.trend_archived}
  />
</div>

<style>
  .stats {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(230px, 1fr));
    gap: var(--space-4);
    margin-bottom: var(--space-5);
  }
</style>
