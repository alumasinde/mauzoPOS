<script lang="ts" generics="T">
  import type { Snippet } from 'svelte';
  import { Button, Icon } from '@mauzo/ui';

  /** Titled, add/remove-capable editor for a repeating child list (barcodes,
   *  variants, components, attributes). Renders `row` for each item; the
   *  parent owns the row shape. `cols` sets grid-template-columns and the
   *  remove button always lands in the trailing column. */
  interface Props {
    title: string;
    items: T[];
    cols?: string;
    onadd: () => void;
    onremove: (i: number) => void;
    row: Snippet<[T]>;
  }
  let { title, items, cols = '1fr 1fr 100px', onadd, onremove, row }: Props = $props();
</script>

<section>
  <header>
    <h4>{title}</h4>
    <Button size="sm" variant="ghost" onclick={onadd}>+ Add</Button>
  </header>
  {#each items as item, i (i)}
    <div class="rowline" style:grid-template-columns="{cols} auto">
      {@render row(item)}
      <Button size="sm" variant="ghost" onclick={() => onremove(i)}><Icon name="x" size={13} /></Button>
    </div>
  {/each}
</section>

<style>
  section {
    margin-top: 1.25rem;
    border-top: 1px solid var(--border, #eee);
    padding-top: 0.75rem;
  }
  header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
  h4 {
    margin: 0 0 0.5rem;
    font-size: 0.8rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-muted, #667);
  }
  .rowline {
    display: grid;
    gap: 0.5rem;
    margin-bottom: 0.5rem;
  }
</style>
