<script lang="ts">
  // Single-value prompt used for quantity and line-discount entry.
  let {
    title,
    value = $bindable(''),
    placeholder = '',
    submitLabel = 'Apply',
    onsubmit,
    oncancel,
  }: {
    title: string;
    value?: string;
    placeholder?: string;
    submitLabel?: string;
    onsubmit: () => void;
    oncancel: () => void;
  } = $props();
</script>

<div class="modal">
  <h3>{title}</h3>
  <input
    class="big"
    bind:value
    {placeholder}
    inputmode="decimal"
    onkeydown={(e) => e.key === 'Enter' && onsubmit()}
  />
  <div class="row2">
    <button class="ghost" onclick={oncancel}>Cancel</button>
    <button class="go" onclick={onsubmit}>{submitLabel}</button>
  </div>
</div>

<style>
  .modal {
    width: min(420px, 92vw);
    background: var(--color-surface);
    border-radius: var(--radius-lg);
    padding: var(--space-5);
    box-shadow: 0 12px 40px rgb(0 0 0 / 0.3);
  }
  h3 {
    margin: 0 0 var(--space-3);
  }
  input {
    width: 100%;
    padding: var(--space-3);
    font: inherit;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    box-sizing: border-box;
  }
  input.big {
    font-size: var(--text-2xl);
    text-align: center;
  }
  .row2 {
    display: flex;
    gap: var(--space-2);
    margin-top: var(--space-3);
  }
  .go,
  .ghost {
    flex: 1;
    padding: var(--space-3) var(--space-4);
    font: inherit;
    border-radius: var(--radius-md);
    cursor: pointer;
  }
  .go {
    font-weight: 700;
    color: #fff;
    background: var(--color-primary);
    border: none;
  }
  .ghost {
    background: none;
    border: 1px solid var(--color-border);
  }
</style>
