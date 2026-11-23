<script lang="ts">
  import type { Product } from '@mauzo/types';
  import { Button } from '@mauzo/ui';

  /** Per-row actions in the products table. Clicks are stopPropagation'd so
   *  they don't retrigger the row's onrowclick edit. */
  interface Props {
    row: Product;
    canManage: boolean;
    canDelete: boolean;
    busy: string | null;
    onedit: (p: Product) => void;
    onarchive: (p: Product) => void;
  }
  let { row, canManage, canDelete, busy, onedit, onarchive }: Props = $props();
</script>

<div class="rowbtns">
  {#if canManage}<Button
      size="sm"
      variant="ghost"
      disabled={busy === row.id}
      onclick={(e) => {
        e.stopPropagation();
        onedit(row);
      }}
    >
      Edit
    </Button>{/if}
  {#if canDelete && row.status === 'active'}<Button
      size="sm"
      variant="ghost"
      disabled={busy === row.id}
      onclick={(e) => {
        e.stopPropagation();
        onarchive(row);
      }}
    >
      Archive
    </Button>{/if}
</div>

<style>
  .rowbtns {
    display: flex;
    gap: 0.25rem;
    justify-content: flex-end;
  }
</style>
