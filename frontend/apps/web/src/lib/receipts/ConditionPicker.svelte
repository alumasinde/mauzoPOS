<script lang="ts">
  // "Show this block": always, when a flag holds, or when a variable has a
  // value — optionally inverted ("!has.customer").
  import type { NamedDoc } from '@mauzo/types';

  interface Props {
    value: string | undefined;
    variables: NamedDoc[];
    conditions: NamedDoc[];
    onset: (next: string | undefined) => void;
  }
  let { value, variables, conditions, onset }: Props = $props();

  const negated = $derived((value ?? '').startsWith('!'));
  const name = $derived((value ?? '').replace(/^!/, ''));

  const emit = (n: string, neg: boolean) => onset(n ? (neg ? '!' : '') + n : undefined);
</script>

<span class="lbl">Show this block</span>
<div class="row">
  <select aria-label="Condition" value={name} onchange={(e) => emit(e.currentTarget.value, negated)}>
    <option value="">Always</option>
    <optgroup label="When">
      {#each conditions as c (c.name)}<option value={c.name}>{c.description}</option>{/each}
    </optgroup>
    <optgroup label="When this has a value">
      {#each variables as v (v.name)}<option value={v.name}>{v.description}</option>{/each}
    </optgroup>
  </select>
  {#if name}
    <label class="chk">
      <input type="checkbox" checked={negated} onchange={(e) => emit(name, e.currentTarget.checked)} />
      Invert (when NOT)
    </label>
  {/if}
</div>

<style>
  .lbl {
    font-size: var(--text-sm);
    font-weight: 600;
    margin-top: var(--space-2);
  }
  .row {
    display: flex;
    gap: var(--space-3);
    align-items: center;
  }
  select {
    width: 100%;
    padding: var(--space-2);
    font: inherit;
    font-size: var(--text-sm);
    color: var(--color-text);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
  }
  .chk {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    font-size: var(--text-sm);
    white-space: nowrap;
  }
</style>
