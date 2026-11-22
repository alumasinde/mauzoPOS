<script lang="ts" generics="T">
  import type { Snippet } from 'svelte';
  import Spinner from './Spinner.svelte';
  import type { Column } from './types';

  interface Props {
    columns: Column[];
    rows: T[];
    rowKey: (row: T) => string | number;
    /** Renders one cell; falls back to row[col.key]. */
    cell?: Snippet<[T, Column]>;
    loading?: boolean;
    empty?: string;
    onrowclick?: (row: T) => void;
  }

  let {
    columns,
    rows,
    rowKey,
    cell,
    loading = false,
    empty = 'No records found.',
    onrowclick,
  }: Props = $props();

  function raw(row: T, key: string): unknown {
    return (row as Record<string, unknown>)[key];
  }
</script>

<div class="wrap">
  <table>
    <thead>
      <tr>
        {#each columns as col (col.key)}
          <th style:text-align={col.align ?? 'left'} style:width={col.width}>{col.label}</th>
        {/each}
      </tr>
    </thead>
    <tbody>
      {#if loading}
        <tr><td colspan={columns.length} class="state"><Spinner /></td></tr>
      {:else if rows.length === 0}
        <tr><td colspan={columns.length} class="state">{empty}</td></tr>
      {:else}
        {#each rows as row (rowKey(row))}
          <tr class:clickable={!!onrowclick} onclick={() => onrowclick?.(row)}>
            {#each columns as col (col.key)}
              <td style:text-align={col.align ?? 'left'}>
                {#if cell}{@render cell(row, col)}{:else}{raw(row, col.key) ?? ''}{/if}
              </td>
            {/each}
          </tr>
        {/each}
      {/if}
    </tbody>
  </table>
</div>

<style>
  .wrap {
    overflow-x: auto;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    background: var(--color-surface);
  }
  table {
    width: 100%;
    border-collapse: collapse;
  }
  th,
  td {
    padding: var(--space-3) var(--space-4);
    border-bottom: 1px solid var(--color-border);
  }
  th {
    font-size: var(--text-sm);
    color: var(--color-text-muted);
    background: var(--color-surface-muted);
    font-weight: 600;
  }
  tbody tr:last-child td {
    border-bottom: none;
  }
  .clickable {
    cursor: pointer;
  }
  .clickable:hover {
    background: var(--color-surface-muted);
  }
  .state {
    text-align: center;
    color: var(--color-text-muted);
    padding: var(--space-6);
  }
</style>
