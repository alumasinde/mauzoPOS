<script lang="ts">
  // The ordered blocks of a layout: select, reorder, duplicate, remove, add.
  import type { ReceiptBlock, ReceiptBlockType } from '@mauzo/types';
  import { Icon } from '@mauzo/ui';
  import { BLOCKS, kindOf, summary } from './blocks';

  interface Props {
    blocks: ReceiptBlock[];
    selected: number;
    onselect: (i: number) => void;
    onchange: () => void;
  }
  let { blocks = $bindable(), selected, onselect, onchange }: Props = $props();

  let adding = $state(false);
  let dragFrom = $state<number | null>(null);

  function move(i: number, by: number) {
    const j = i + by;
    if (j < 0 || j >= blocks.length) return;
    const next = [...blocks];
    [next[i], next[j]] = [next[j]!, next[i]!];
    blocks = next;
    onselect(j);
    onchange();
  }

  function drop(to: number) {
    if (dragFrom === null || dragFrom === to) return;
    const next = [...blocks];
    const [b] = next.splice(dragFrom, 1);
    next.splice(to, 0, b!);
    blocks = next;
    onselect(to);
    dragFrom = null;
    onchange();
  }

  function duplicate(i: number) {
    const next = [...blocks];
    next.splice(i + 1, 0, structuredClone($state.snapshot(blocks[i]!)) as ReceiptBlock);
    blocks = next;
    onselect(i + 1);
    onchange();
  }

  function remove(i: number) {
    blocks = blocks.filter((_, j) => j !== i);
    onselect(Math.min(i, blocks.length - 1));
    onchange();
  }

  function add(type: ReceiptBlockType) {
    const at = selected >= 0 ? selected + 1 : blocks.length;
    const next = [...blocks];
    next.splice(at, 0, kindOf(type).make());
    blocks = next;
    onselect(at);
    adding = false;
    onchange();
  }
</script>

<ol class="list" aria-label="Receipt blocks">
  {#each blocks as b, i (i)}
    <li
      class:sel={i === selected}
      class:over={dragFrom !== null && dragFrom !== i}
      draggable="true"
      ondragstart={() => (dragFrom = i)}
      ondragover={(e) => e.preventDefault()}
      ondrop={() => drop(i)}
      ondragend={() => (dragFrom = null)}
    >
      <button class="pick" onclick={() => onselect(i)} aria-current={i === selected ? 'true' : undefined}>
        <span class="grip" aria-hidden="true">⋮⋮</span>
        <Icon name={kindOf(b.type).icon} size={16} />
        <span class="txt">
          <strong>{kindOf(b.type).label}</strong>
          <small>{summary(b)}</small>
        </span>
        {#if b.if}<span class="cond" title="Shown only {b.if}">if</span>{/if}
      </button>
      <span class="tools">
        <button aria-label="Move up" disabled={i === 0} onclick={() => move(i, -1)}>↑</button>
        <button aria-label="Move down" disabled={i === blocks.length - 1} onclick={() => move(i, 1)}>
          ↓
        </button>
        <button aria-label="Duplicate" onclick={() => duplicate(i)}>⧉</button>
        <button aria-label="Remove" onclick={() => remove(i)}><Icon name="x" size={14} /></button>
      </span>
    </li>
  {/each}
</ol>

{#if adding}
  <div class="palette" role="menu" aria-label="Add block">
    {#each BLOCKS as k (k.type)}
      <button role="menuitem" onclick={() => add(k.type)} title={k.hint}>
        <Icon name={k.icon} size={16} />
        {k.label}
      </button>
    {/each}
    <button class="cancel" onclick={() => (adding = false)}>Cancel</button>
  </div>
{:else}
  <button class="add" onclick={() => (adding = true)}>+ Add block</button>
{/if}

<style>
  .list {
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    gap: 4px;
  }
  li {
    min-width: 0;
    display: flex;
    align-items: stretch;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    background: var(--color-surface);
  }
  li.sel {
    border-color: var(--color-primary);
    box-shadow: 0 0 0 1px var(--color-primary);
  }
  li.over:hover {
    border-style: dashed;
  }
  .pick {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: 6px var(--space-2);
    background: none;
    border: 0;
    font: inherit;
    color: var(--color-text);
    text-align: left;
    cursor: pointer;
  }
  .grip {
    color: var(--color-text-muted);
    cursor: grab;
    font-size: 11px;
    letter-spacing: -2px;
  }
  .txt {
    display: flex;
    flex-direction: column;
    min-width: 0;
    line-height: 1.2;
  }
  .txt strong {
    font-size: var(--text-sm);
  }
  .txt small {
    color: var(--color-text-muted);
    font-size: var(--text-xs);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .cond {
    margin-left: auto;
    font-size: 10px;
    font-weight: 700;
    padding: 1px 6px;
    border-radius: 999px;
    background: var(--color-surface-muted);
    color: var(--color-primary);
  }
  .tools {
    display: flex;
    align-items: center;
    padding-right: 4px;
  }
  .tools button {
    display: grid;
    place-items: center;
    width: 24px;
    height: 24px;
    background: none;
    border: 0;
    border-radius: 4px;
    color: var(--color-text-muted);
    cursor: pointer;
    font-size: 13px;
  }
  .tools button:hover:not(:disabled) {
    background: var(--color-surface-muted);
    color: var(--color-text);
  }
  .tools button:disabled {
    opacity: 0.3;
    cursor: default;
  }
  .add,
  .palette button {
    font: inherit;
    font-size: var(--text-sm);
    cursor: pointer;
  }
  .add {
    width: 100%;
    margin-top: var(--space-2);
    padding: var(--space-2);
    border: 1px dashed var(--color-border);
    border-radius: var(--radius-sm);
    background: none;
    color: var(--color-primary);
    font-weight: 600;
  }
  .palette {
    margin-top: var(--space-2);
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 4px;
  }
  .palette button {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: 6px var(--space-2);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    background: var(--color-surface);
    color: var(--color-text);
  }
  .palette button:hover {
    border-color: var(--color-primary);
  }
  .palette .cancel {
    grid-column: 1 / -1;
    justify-content: center;
    color: var(--color-text-muted);
  }
</style>
