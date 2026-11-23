<script lang="ts">
  // Single-series area chart with a y-axis, x labels and a hover tooltip.
  // Dependency-free SVG; colours come from tokens.
  import type { AreaPoint } from './types';

  interface Props {
    points: AreaPoint[];
    /** Formats y values for the axis and tooltip. */
    format?: (v: number) => string;
    height?: number;
    label: string;
  }
  let { points, format = (v) => String(v), height = 240, label }: Props = $props();

  const W = 640;
  const PAD = { top: 12, right: 12, bottom: 28, left: 56 };
  let hover = $state<number | null>(null);
  const uid = $props.id();
  const fillId = `area-fill-${uid}`;

  const max = $derived(niceMax(Math.max(0, ...points.map((p) => p.value))));
  const plotW = W - PAD.left - PAD.right;
  const plotH = $derived(height - PAD.top - PAD.bottom);
  const x = (i: number) => PAD.left + (points.length < 2 ? plotW / 2 : (i * plotW) / (points.length - 1));
  const y = (v: number) => PAD.top + plotH - (max ? (v / max) * plotH : 0);

  const line = $derived(points.map((p, i) => `${i ? 'L' : 'M'}${x(i)} ${y(p.value)}`).join(' '));
  const area = $derived(
    points.length ? `${line} L${x(points.length - 1)} ${PAD.top + plotH} L${x(0)} ${PAD.top + plotH} Z` : '',
  );
  const ticks = $derived([0, 0.25, 0.5, 0.75, 1].map((f) => f * max));
  // Show at most ~8 x labels.
  const every = $derived(Math.max(1, Math.ceil(points.length / 8)));

  function niceMax(v: number): number {
    if (v <= 0) return 0;
    const mag = 10 ** Math.floor(Math.log10(v));
    const n = v / mag;
    return (n <= 1 ? 1 : n <= 2 ? 2 : n <= 4 ? 4 : n <= 5 ? 5 : 10) * mag;
  }

  function move(e: PointerEvent) {
    const svg = e.currentTarget as SVGSVGElement;
    const r = svg.getBoundingClientRect();
    const px = ((e.clientX - r.left) / r.width) * W;
    if (points.length < 2) return void (hover = points.length ? 0 : null);
    const i = Math.round(((px - PAD.left) / plotW) * (points.length - 1));
    hover = Math.min(points.length - 1, Math.max(0, i));
  }
</script>

<svg
  viewBox="0 0 {W} {height}"
  class="chart"
  role="img"
  aria-label={label}
  onpointermove={move}
  onpointerleave={() => (hover = null)}
>
  <defs>
    <linearGradient id={fillId} x1="0" y1="0" x2="0" y2="1">
      <stop offset="0%" stop-color="var(--chart-1)" stop-opacity="0.45" />
      <stop offset="100%" stop-color="var(--chart-1)" stop-opacity="0.02" />
    </linearGradient>
  </defs>
  {#each ticks as t (t)}
    <line x1={PAD.left} x2={W - PAD.right} y1={y(t)} y2={y(t)} class="grid" />
    <text x={PAD.left - 8} y={y(t) + 4} text-anchor="end" class="axis">{format(t)}</text>
  {/each}
  {#each points as p, i (i)}
    {#if i % every === 0 || i === points.length - 1}
      <text x={x(i)} y={height - 8} text-anchor="middle" class="axis">{p.label}</text>
    {/if}
  {/each}
  {#if points.length}
    <path d={area} fill="url(#{fillId})" />
    <path d={line} fill="none" stroke="var(--chart-1)" stroke-width="2.5" stroke-linejoin="round" />
  {/if}
  {#if hover !== null && points[hover]}
    {@const pt = points[hover]!}
    {@const hx = x(hover)}
    {@const hy = y(pt.value)}
    {@const tx = Math.min(Math.max(hx - 60, PAD.left), W - PAD.right - 120)}
    <line x1={hx} x2={hx} y1={PAD.top} y2={PAD.top + plotH} class="cursor" />
    <circle cx={hx} cy={hy} r="5" fill="var(--chart-1)" stroke="var(--color-surface)" stroke-width="2" />
    <g transform="translate({tx} {Math.max(PAD.top, hy - 58)})">
      <rect width="120" height="44" rx="8" class="tip" />
      <text x="12" y="18" class="tip-label">{pt.label}</text>
      <text x="12" y="35" class="tip-value">{format(pt.value)}</text>
    </g>
  {/if}
</svg>

<style>
  .chart {
    display: block;
    width: 100%;
    height: auto;
    touch-action: none;
  }
  .grid {
    stroke: var(--color-border);
    stroke-dasharray: 3 4;
  }
  .axis {
    fill: var(--color-text-muted);
    font-size: 11px;
  }
  .cursor {
    stroke: var(--color-text-muted);
    stroke-dasharray: 3 3;
  }
  .tip {
    fill: var(--color-surface-raised, var(--color-surface));
    stroke: var(--color-border);
  }
  .tip-label {
    fill: var(--color-text-muted);
    font-size: 11px;
  }
  .tip-value {
    fill: var(--color-text);
    font-size: 13px;
    font-weight: 700;
  }
</style>
