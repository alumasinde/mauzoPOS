<script lang="ts">
  // The tenant's saved layouts.
  import type { ReceiptTemplate } from '@mauzo/types';
  import { Button } from '@mauzo/ui';
  import { formatDate } from '@mauzo/utils';

  interface Props {
    templates: ReceiptTemplate[];
    currentId: string | null;
    onopen: (t: ReceiptTemplate) => void;
    onnew: () => void;
  }
  let { templates, currentId, onopen, onnew }: Props = $props();
</script>

<div class="head">
  <h2>Layouts</h2>
  <Button size="sm" onclick={onnew}>New</Button>
</div>
{#if templates.length === 0}
  <p class="muted">
    No custom layouts yet — tills print the default for your business type. Create one from a starting layout.
  </p>
{/if}
<ul>
  {#each templates as t (t.id)}
    <li>
      <button class:on={t.id === currentId} onclick={() => onopen(t)}>
        <strong>{t.name}</strong>
        <small>{t.paper_width} mm · v{t.version} · {formatDate(t.updated_at)}</small>
      </button>
    </li>
  {/each}
</ul>

<style>
  .head {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
  h2 {
    margin: 0;
    font-size: var(--text-lg);
  }
  .muted {
    color: var(--color-text-muted);
    font-size: var(--text-sm);
  }
  ul {
    list-style: none;
    margin: var(--space-3) 0 0;
    padding: 0;
    display: grid;
    gap: 4px;
  }
  button {
    width: 100%;
    display: flex;
    flex-direction: column;
    text-align: left;
    padding: var(--space-2);
    background: none;
    border: 1px solid transparent;
    border-radius: var(--radius-sm);
    font: inherit;
    color: var(--color-text);
    cursor: pointer;
  }
  button:hover {
    background: var(--color-surface-muted);
  }
  button.on {
    border-color: var(--color-primary);
    background: color-mix(in srgb, var(--color-primary) 10%, transparent);
  }
  small {
    color: var(--color-text-muted);
    font-size: var(--text-xs);
  }
</style>
