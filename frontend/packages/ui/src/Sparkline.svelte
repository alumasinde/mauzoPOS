<script lang="ts">
  // A tiny trend line for KPI cards. Inherits its colour from `currentColor`.
  // With fewer than two points it draws a flat baseline.
  interface Props {
    values: number[];
    width?: number;
    height?: number;
    label?: string;
  }
  let { values, width = 160, height = 40, label }: Props = $props();

  const path = $derived.by(() => {
    if (values.length < 2) return `M0 ${height - 2} H${width}`;
    const min = Math.min(...values);
    const max = Math.max(...values);
    const span = max - min || 1;
    const step = width / (values.length - 1);
    return values
      .map(
        (v, i) =>
          `${i ? 'L' : 'M'}${(i * step).toFixed(1)} ${(height - 3 - ((v - min) / span) * (height - 6)).toFixed(1)}`,
      )
      .join(' ');
  });
</script>

<svg
  viewBox="0 0 {width} {height}"
  preserveAspectRatio="none"
  class="spark"
  role={label ? 'img' : undefined}
  aria-label={label}
  aria-hidden={label ? undefined : 'true'}
>
  <path d={path} fill="none" stroke="currentColor" stroke-width="2" vector-effect="non-scaling-stroke" />
</svg>

<style>
  .spark {
    display: block;
    width: 100%;
    height: 40px;
  }
</style>
