<script lang="ts" module>
  export type StatTone = 'blue' | 'green' | 'orange' | 'purple' | 'teal' | 'red';
</script>

<script lang="ts">
  // Compact KPI card: tinted icon tile, label, headline value, delta chip and
  // an inline sparkline. All data comes from the caller — never fabricated.
  import Icon from './Icon.svelte';
  import Sparkline from './Sparkline.svelte';

  interface Props {
    label: string;
    value: string;
    icon: string;
    tone?: StatTone;
    /** Percentage change, e.g. 2.5 or -1.2; omit to hide the chip. */
    delta?: number | null;
    /** Text after the chip, e.g. "vs last month" or "this month". */
    deltaLabel?: string;
    /** Sparkline points; empty shows a flat line. */
    trend?: number[];
  }
  let { label, value, icon, tone = 'blue', delta = null, deltaLabel = '', trend = [] }: Props = $props();
</script>

<article class="card {tone}" aria-label={label}>
  <div class="body">
    <span class="tile" aria-hidden="true"><Icon name={icon} size={22} strokeWidth={1.8} /></span>
    <div class="txt">
      <p class="label">{label}</p>
      <p class="value">{value}</p>
      {#if delta !== null}
        <p class="delta" class:neg={delta < 0}>
          <Icon name={delta >= 0 ? 'arrow-up' : 'arrow-down'} size={12} strokeWidth={2.6} />
          {Math.abs(delta).toFixed(1)}%{#if deltaLabel}<span class="cmp">{deltaLabel}</span>{/if}
        </p>
      {/if}
    </div>
  </div>
  <div class="spark" aria-hidden="true"><Sparkline values={trend} width={120} height={44} /></div>
</article>

<style>
  .card {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-3);
    padding: var(--space-4);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-lg);
    min-width: 0;
    overflow: hidden;
  }
  .body {
    display: flex;
    align-items: flex-start;
    gap: var(--space-3);
    min-width: 0;
  }
  .tile {
    display: grid;
    place-items: center;
    width: 42px;
    height: 42px;
    border-radius: var(--radius-md);
    flex: none;
  }
  .blue .tile {
    background: color-mix(in srgb, var(--chart-1) 16%, transparent);
    color: var(--chart-1);
  }
  .green .tile {
    background: color-mix(in srgb, var(--chart-2) 16%, transparent);
    color: var(--chart-2);
  }
  .orange .tile {
    background: color-mix(in srgb, var(--chart-4) 16%, transparent);
    color: var(--chart-4);
  }
  .purple .tile {
    background: color-mix(in srgb, var(--chart-3) 16%, transparent);
    color: var(--chart-3);
  }
  .teal .tile {
    background: color-mix(in srgb, var(--chart-5) 16%, transparent);
    color: var(--chart-5);
  }
  .red .tile {
    background: color-mix(in srgb, var(--color-danger) 16%, transparent);
    color: var(--color-danger);
  }
  .txt {
    min-width: 0;
  }
  .label {
    margin: 1px 0 0;
    font-size: var(--text-sm);
    color: var(--color-text-muted);
    white-space: nowrap;
  }
  .value {
    margin: 2px 0;
    font-size: var(--text-2xl);
    font-weight: 800;
    letter-spacing: -0.02em;
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
  }
  .delta {
    margin: 0;
    display: inline-flex;
    align-items: center;
    gap: 3px;
    font-size: var(--text-xs);
    font-weight: 700;
    color: var(--color-success);
  }
  .delta.neg {
    color: var(--color-danger);
  }
  .cmp {
    font-weight: 500;
    color: var(--color-text-muted);
    margin-left: 3px;
  }
  .spark {
    flex: none;
    align-self: flex-end;
    margin-bottom: 2px;
  }
  .blue .spark {
    color: var(--chart-1);
  }
  .green .spark {
    color: var(--chart-2);
  }
  .orange .spark {
    color: var(--chart-4);
  }
  .purple .spark {
    color: var(--chart-3);
  }
  .teal .spark {
    color: var(--chart-5);
  }
  .red .spark {
    color: var(--color-danger);
  }
  @media (max-width: 720px) {
    .spark {
      display: none;
    }
  }
</style>
