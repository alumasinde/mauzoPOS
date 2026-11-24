<script lang="ts">
  // Shown when a scanned barcode is not in the local catalog.
  let {
    code,
    onsync,
    onclose,
  }: {
    code: string;
    onsync: () => void;
    onclose: () => void;
  } = $props();
</script>

<div class="modal">
  <h3>Unknown barcode</h3>
  <p class="muted"><code>{code}</code> isn't in the catalog. Check the label or sync the catalog.</p>
  <div class="row2">
    <button class="ghost" onclick={onclose}>Dismiss</button>
    <button
      class="go"
      onclick={() => {
        onsync();
        onclose();
      }}
    >
      Sync catalog
    </button>
  </div>
</div>

<style>
  h3 {
    margin: 0 0 var(--space-3);
  }
  .muted {
    color: var(--color-text-muted);
  }
  .row2 {
    display: flex;
    gap: var(--space-2);
    margin-top: var(--space-3);
  }
  .go,
  .ghost {
    flex: 1;
    font: inherit;
    border: none;
    border-radius: var(--radius-md);
    cursor: pointer;
  }
  .go {
    padding: var(--space-3) var(--space-4);
    font-weight: 700;
    color: #fff;
    background: var(--color-primary);
  }
  .ghost {
    padding: var(--space-2) var(--space-4);
    background: none;
    border: 1px solid var(--color-border);
  }
</style>
