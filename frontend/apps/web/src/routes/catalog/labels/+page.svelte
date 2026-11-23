<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/state';
  import { ApiError } from '@mauzo/api-client';
  import type { Label } from '@mauzo/types';
  import { Button, toast, PageHeader } from '@mauzo/ui';
  import { api } from '#lib/api.ts';

  let labels = $state<Label[]>([]);
  let loading = $state(true);
  let copies = $state(24);

  /** Renders a Code 128 module pattern as SVG <rect>s — no font needed. */
  function bars(pattern: boolean[]): { x: number; w: number }[] {
    const out: { x: number; w: number }[] = [];
    for (let i = 0; i < pattern.length; i++) {
      if (!pattern[i]) continue;
      let j = i;
      while (j + 1 < pattern.length && pattern[j + 1]) j++;
      out.push({ x: i, w: j - i + 1 });
      i = j;
    }
    return out;
  }

  async function load() {
    loading = true;
    try {
      const products = page.url.searchParams.get('products') ?? '';
      labels = await api.get<Label[]>(`/api/v1/t/catalog/labels?products=${products}&copies=${copies}`);
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Failed to load labels');
    } finally {
      loading = false;
    }
  }

  onMount(load);
</script>

<svelte:head><title>Print labels — Mauzo</title></svelte:head>

<PageHeader title="Shelf labels" sub="Print price labels and barcodes for the shelf." icon="barcode">
  {#snippet actions()}
    <label class="no-print">
      Copies each <input type="number" min="1" max="100" bind:value={copies} onchange={load} />
    </label>
    <Button onclick={() => window.print()}>Print</Button>
  {/snippet}
</PageHeader>

{#if loading}
  <p class="no-print">Loading…</p>
{:else if labels.length === 0}
  <p class="no-print">No labels — choose products first.</p>
{/if}

<div class="sheet">
  {#each labels as l, i (i)}
    <div class="label">
      <div class="name">{l.name}</div>
      <svg viewBox="0 0 {l.pattern.length} 40" preserveAspectRatio="none">
        {#each bars(l.pattern) as b (b.x)}
          <rect x={b.x} y="0" width={b.w} height="40" fill="#000" />
        {/each}
      </svg>
      <div class="code">{l.barcode}</div>
      <div class="price">KES {l.price}</div>
    </div>
  {/each}
</div>

<style>
  label input {
    width: 70px;
    padding: 0.35rem;
    border: 1px solid var(--border, #d8dce3);
    border-radius: 6px;
    font: inherit;
  }
  .sheet {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
    gap: 6px;
  }
  .label {
    border: 1px dashed #999;
    padding: 6px;
    display: flex;
    flex-direction: column;
    gap: 3px;
    page-break-inside: avoid;
    background: #fff;
  }
  .name {
    font-size: 0.75rem;
    font-weight: 600;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  svg {
    width: 100%;
    height: 34px;
  }
  .code {
    font-size: 0.65rem;
    letter-spacing: 0.1em;
    text-align: center;
  }
  .price {
    font-size: 0.9rem;
    font-weight: 700;
    text-align: right;
  }
  @media print {
    .no-print {
      display: none;
    }
    .sheet {
      grid-template-columns: repeat(4, 1fr);
    }
    :global(body) {
      margin: 0;
    }
  }
</style>
