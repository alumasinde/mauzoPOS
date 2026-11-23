<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    open: boolean;
    title: string;
    /** Prevent closing via Esc/backdrop (e.g. while saving). */
    locked?: boolean;
    onclose?: () => void;
    /** md (default, 560px) or lg (800px) for dense editors. */
    size?: 'md' | 'lg';
    children: Snippet;
    footer?: Snippet;
  }

  let { open = $bindable(), title, locked = false, onclose, size = 'md', children, footer }: Props = $props();
  let dialog = $state<HTMLDialogElement>();

  $effect(() => {
    if (!dialog) return;
    if (open && !dialog.open) dialog.showModal();
    else if (!open && dialog.open) dialog.close();
  });

  function close() {
    if (locked) return;
    open = false;
    onclose?.();
  }
</script>

<dialog
  bind:this={dialog}
  class={size}
  aria-labelledby="modal-title"
  oncancel={(e) => {
    e.preventDefault();
    close();
  }}
  onclick={(e) => e.target === dialog && close()}
>
  <div class="panel">
    <header>
      <h2 id="modal-title">{title}</h2>
      <button class="x" aria-label="Close" onclick={close} disabled={locked}>×</button>
    </header>
    <div class="body">{@render children()}</div>
    {#if footer}<footer>{@render footer()}</footer>{/if}
  </div>
</dialog>

<style>
  dialog {
    padding: 0;
    border: none;
    border-radius: var(--radius-lg);
    background: var(--color-surface);
    color: var(--color-text);
    box-shadow: var(--shadow-lg);
    width: min(560px, calc(100vw - 32px));
    overflow-x: hidden;
  }
  dialog.lg {
    width: min(800px, calc(100vw - 32px));
  }
  dialog::backdrop {
    background: rgb(8 12 24 / 0.55);
  }
  header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: var(--space-4) var(--space-5);
    border-bottom: 1px solid var(--color-border);
  }
  h2 {
    margin: 0;
    font-size: var(--text-lg);
  }
  .x {
    border: none;
    background: none;
    font-size: var(--text-2xl);
    line-height: 1;
    cursor: pointer;
    color: var(--color-text-muted);
  }
  .body {
    padding: var(--space-5);
  }
  footer {
    display: flex;
    justify-content: flex-end;
    gap: var(--space-2);
    padding: var(--space-4) var(--space-5);
    border-top: 1px solid var(--color-border);
  }
</style>
