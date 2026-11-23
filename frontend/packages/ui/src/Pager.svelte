<script lang="ts">
  interface Props {
    page: number;
    perPage: number;
    total: number;
    onpage: (page: number) => void;
  }

  let { page, perPage, total, onpage }: Props = $props();
  const pages = $derived(Math.max(1, Math.ceil(total / perPage)));
</script>

{#if pages > 1}
  <nav class="pager" aria-label="Pagination">
    <button disabled={page <= 1} onclick={() => onpage(page - 1)}>Previous</button>
    <span>Page {page} of {pages}</span>
    <button disabled={page >= pages} onclick={() => onpage(page + 1)}>Next</button>
  </nav>
{/if}

<style>
  .pager {
    display: flex;
    align-items: center;
    gap: var(--space-4);
    justify-content: center;
    margin-top: var(--space-4);
  }
  button {
    border: 1px solid var(--color-border);
    background: var(--color-surface);
    color: var(--color-text);
    border-radius: var(--radius-sm);
    padding: var(--space-2) var(--space-4);
    font: inherit;
    cursor: pointer;
  }
  button:disabled {
    opacity: 0.5;
    cursor: default;
  }
</style>
