<script lang="ts">
  // Live preview of the layout being edited, with sample or real sale data.
  import type { ReceiptOutput, Sale } from '@mauzo/types';
  import { Icon } from '@mauzo/ui';
  import ReceiptPaper from './ReceiptPaper.svelte';

  interface Props {
    output: ReceiptOutput | null;
    problems: string[];
    logoUrl: string | null;
    recent: Sale[];
    /** Sale public id; "" = sample data. */
    source: string;
    onsource: (id: string) => void;
  }
  let { output, problems, logoUrl, recent, source, onsource }: Props = $props();
</script>

<div class="head">
  <h2>Preview</h2>
  <select aria-label="Preview data" value={source} onchange={(e) => onsource(e.currentTarget.value)}>
    <option value="">Sample sale</option>
    {#each recent as s (s.id)}
      <option value={s.id}>Receipt #{s.receipt_no} · {s.total}</option>
    {/each}
  </select>
</div>
{#if problems.length}
  <ul class="problems" role="alert">
    {#each problems as p, i (i)}<li><Icon name="alert" size={14} /> {p}</li>{/each}
  </ul>
{/if}
{#if output}
  <div class="wrap"><ReceiptPaper {output} {logoUrl} /></div>
{/if}
<p class="muted">
  Printed exactly like this by the till. QR codes are drawn by the printer itself.
  {#if !logoUrl}Upload a logo under “Logo &amp; details”.{/if}
</p>

<style>
  .head {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: var(--space-2);
  }
  h2 {
    margin: 0;
    font-size: var(--text-lg);
  }
  select {
    padding: var(--space-1) var(--space-2);
    font: inherit;
    font-size: var(--text-sm);
    color: var(--color-text);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    max-width: 60%;
  }
  .wrap {
    display: flex;
    justify-content: center;
    padding: var(--space-4) 0;
    background: var(--color-surface-muted);
    border-radius: var(--radius-md);
    margin: var(--space-3) 0;
    max-height: 70vh;
    overflow: auto;
  }
  .problems {
    list-style: none;
    margin: var(--space-3) 0 0;
    padding: var(--space-2) var(--space-3);
    border-radius: var(--radius-sm);
    background: color-mix(in srgb, var(--color-danger) 12%, transparent);
    color: var(--color-danger);
    font-size: var(--text-sm);
    display: grid;
    gap: 2px;
  }
  .problems li {
    display: flex;
    gap: var(--space-2);
    align-items: center;
  }
  .muted {
    color: var(--color-text-muted);
    font-size: var(--text-sm);
    margin: 0;
  }
</style>
