<script lang="ts">
  // Donut with a centre caption and a legend. Slice colours cycle through the
  // --chart-N tokens unless a slice names its own.
  import type { DonutSlice } from './types';

  interface Props {
    slices: DonutSlice[];
    centerValue?: string;
    centerLabel?: string;
    label: string;
  }
  let { slices, centerValue, centerLabel, label }: Props = $props();

  const R = 70;
  const C = 2 * Math.PI * R;
  const total = $derived(slices.reduce((s, x) => s + Math.max(0, x.value), 0));
  const color = (s: DonutSlice, i: number) => s.color ?? `var(--chart-${(i % 5) + 1})`;
  const arcs = $derived.by(() => {
    let offset = 0;
    return slices.map((s, i) => {
      const len = total ? (Math.max(0, s.value) / total) * C : 0;
      const a = { len, offset, color: color(s, i) };
      offset += len;
      return a;
    });
  });
  const pct = (v: number) => (total ? `${((v / total) * 100).toFixed(1)}%` : '—');
</script>

<div class="donut">
  <svg viewBox="0 0 180 180" role="img" aria-label={label}>
    <circle cx="90" cy="90" r={R} fill="none" stroke="var(--color-surface-muted)" stroke-width="20" />
    {#each arcs as a, i (i)}
      {#if a.len > 0}
        <circle
          cx="90"
          cy="90"
          r={R}
          fill="none"
          stroke={a.color}
          stroke-width="20"
          stroke-dasharray="{a.len} {C - a.len}"
          stroke-dashoffset={-a.offset}
          transform="rotate(-90 90 90)"
        />
      {/if}
    {/each}
    {#if centerValue}
      <text x="90" y="88" text-anchor="middle" class="cv">{centerValue}</text>
    {/if}
    {#if centerLabel}
      <text x="90" y="108" text-anchor="middle" class="cl">{centerLabel}</text>
    {/if}
  </svg>
  <ul>
    {#each slices as s, i (s.label)}
      <li>
        <span class="sw" style:background={color(s, i)}></span>
        <span class="name">{s.label}</span>
        <span class="pct">{pct(s.value)}</span>
      </li>
    {/each}
  </ul>
</div>

<style>
  .donut {
    display: flex;
    align-items: center;
    gap: var(--space-4);
    flex-wrap: wrap;
    justify-content: center;
  }
  svg {
    width: 170px;
    height: 170px;
    flex: none;
  }
  .cv {
    fill: var(--color-text);
    font-size: 17px;
    font-weight: 700;
  }
  .cl {
    fill: var(--color-text-muted);
    font-size: 11px;
  }
  ul {
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
    gap: var(--space-3);
    min-width: 110px;
  }
  li {
    display: grid;
    grid-template-columns: 12px 1fr;
    column-gap: var(--space-2);
    align-items: center;
  }
  .sw {
    width: 12px;
    height: 12px;
    border-radius: 50%;
  }
  .name {
    font-weight: 600;
    font-size: var(--text-sm);
  }
  .pct {
    grid-column: 2;
    color: var(--color-text-muted);
    font-size: var(--text-sm);
  }
</style>
