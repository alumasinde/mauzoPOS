<script lang="ts">
  import { Icon, Sparkline } from '@mauzo/ui';
  import type { KpiTone } from './types';

  interface Props {
    title: string;
    icon: string;
    tone: KpiTone;
    /** Formatted headline value; null while the module has no data source. */
    value: string | null;
    /** Percentage change vs the previous period, e.g. 12.5 or -3. */
    delta?: number | null;
    compare?: string;
    trend?: number[];
    /** Shown instead of the delta when value is null. */
    pendingNote?: string;
  }
  let { title, icon, tone, value, delta = null, compare = '', trend = [], pendingNote }: Props = $props();
</script>

<article class="kpi {tone}" aria-label={title}>
  <div class="top">
    <span class="ico"><Icon name={icon} size={30} strokeWidth={1.7} /></span>
    <div class="text">
      <h3>{title}</h3>
      <p class="value">{value ?? '—'}</p>
      {#if value === null}
        <p class="sub">{pendingNote ?? 'No data yet'}</p>
      {:else if delta !== null}
        <p class="sub">
          <Icon name={delta >= 0 ? 'arrow-up' : 'arrow-down'} size={13} strokeWidth={2.4} />
          {Math.abs(delta).toFixed(1)}% {compare}
        </p>
      {/if}
    </div>
  </div>
  <div class="spark" class:flat={value === null}><Sparkline values={trend} /></div>
</article>

<style>
  .kpi {
    position: relative;
    overflow: hidden;
    border-radius: var(--radius-lg);
    padding: var(--space-4) var(--space-4) var(--space-2);
    color: #fff;
    box-shadow: var(--shadow-md);
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    min-height: 150px;
  }
  .blue {
    background: var(--kpi-blue);
  }
  .green {
    background: var(--kpi-green);
  }
  .purple {
    background: var(--kpi-purple);
  }
  .teal {
    background: var(--kpi-teal);
  }
  .orange {
    background: var(--kpi-orange);
  }
  .top {
    display: flex;
    gap: var(--space-3);
    align-items: flex-start;
  }
  .ico {
    opacity: 0.95;
    padding-top: 2px;
  }
  .text {
    min-width: 0;
  }
  h3 {
    margin: 0;
    font-size: var(--text-sm);
    font-weight: 600;
    opacity: 0.95;
  }
  .value {
    margin: 2px 0;
    font-size: 1.5rem;
    font-weight: 800;
    letter-spacing: -0.01em;
    white-space: nowrap;
    font-variant-numeric: tabular-nums;
  }
  .sub {
    margin: 0;
    display: flex;
    align-items: center;
    gap: 4px;
    font-size: var(--text-xs);
    opacity: 0.85;
  }
  .spark {
    color: rgb(255 255 255 / 0.7);
    margin: var(--space-2) calc(-1 * var(--space-4)) 0;
  }
  .spark.flat {
    color: rgb(255 255 255 / 0.25);
  }
</style>
