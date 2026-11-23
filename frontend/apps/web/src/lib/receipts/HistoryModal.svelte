<script lang="ts">
  // Saved versions of a layout; loading one puts it in the editor (saving
  // then makes it the newest version — history is never rewritten).
  import type { ReceiptTemplateVersion } from '@mauzo/types';
  import { Button, Modal } from '@mauzo/ui';
  import { formatDate } from '@mauzo/utils';

  interface Props {
    open: boolean;
    versions: ReceiptTemplateVersion[];
    current: number;
    onrestore: (v: ReceiptTemplateVersion) => void;
  }
  let { open = $bindable(), versions, current, onrestore }: Props = $props();
</script>

<Modal bind:open title="Layout history">
  <ul>
    {#each versions as v (v.version)}
      <li>
        <span>
          <strong>Version {v.version}</strong>{#if v.version === current}<em> (current)</em>{/if}
          <small>{formatDate(v.created_at)}{v.author ? ` · ${v.author}` : ''} · {v.paper_width} mm</small>
        </span>
        {#if v.version !== current}
          <Button size="sm" variant="secondary" onclick={() => onrestore(v)}>Load</Button>
        {/if}
      </li>
    {/each}
  </ul>
  {#snippet footer()}
    <Button variant="secondary" onclick={() => (open = false)}>Close</Button>
  {/snippet}
</Modal>

<style>
  ul {
    list-style: none;
    margin: 0;
    padding: 0;
  }
  li {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: var(--space-2) 0;
    border-bottom: 1px solid var(--color-border);
  }
  span {
    display: flex;
    flex-direction: column;
  }
  small {
    color: var(--color-text-muted);
  }
</style>
