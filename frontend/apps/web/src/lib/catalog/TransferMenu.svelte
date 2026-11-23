<script lang="ts">
  import { Button, Icon, toast } from '@mauzo/ui';
  import { onClickOutside } from '#lib/shell/outside.ts';
  import { exportProducts } from '#lib/catalog/transfer.ts';

  async function doExport(format: 'csv' | 'xlsx') {
    exportOpen = false;
    try {
      await exportProducts(format);
    } catch {
      toast.error('Export failed');
    }
  }

  let {
    onimport,
    onlabels,
  }: {
    onimport: () => void;
    onlabels: () => void;
  } = $props();

  let importOpen = $state(false);
  let exportOpen = $state(false);
</script>

<span class="menu" use:onClickOutside={() => (importOpen = false)}>
  <Button variant="secondary" onclick={() => (importOpen = !importOpen)}>
    Import <Icon name="chevron-down" size={14} />
  </Button>
  {#if importOpen}
    <ul class="drop">
      <li>
        <button
          onclick={() => {
            importOpen = false;
            onimport();
          }}
        >
          CSV or Excel file
        </button>
      </li>
    </ul>
  {/if}
</span>
<span class="menu" use:onClickOutside={() => (exportOpen = false)}>
  <Button variant="secondary" onclick={() => (exportOpen = !exportOpen)}>
    Export <Icon name="chevron-down" size={14} />
  </Button>
  {#if exportOpen}
    <ul class="drop">
      <li><button onclick={() => doExport('csv')}>CSV</button></li>
      <li><button onclick={() => doExport('xlsx')}>Excel (.xlsx)</button></li>
      <li>
        <button
          onclick={() => {
            exportOpen = false;
            onlabels();
          }}
        >
          Barcode labels
        </button>
      </li>
    </ul>
  {/if}
</span>

<style>
  .menu {
    position: relative;
    display: inline-flex;
  }
  .drop {
    position: absolute;
    top: calc(100% + 4px);
    right: 0;
    z-index: 30;
    margin: 0;
    padding: var(--space-1);
    list-style: none;
    min-width: 170px;
    background: var(--color-surface-raised);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    box-shadow: var(--shadow-lg);
  }
  .drop button {
    display: block;
    width: 100%;
    text-align: left;
    padding: 8px 10px;
    border: 0;
    border-radius: var(--radius-sm);
    background: none;
    color: var(--color-text);
    font: inherit;
    font-size: var(--text-sm);
    cursor: pointer;
  }
  .drop button:hover {
    background: var(--color-surface-muted);
  }
</style>
