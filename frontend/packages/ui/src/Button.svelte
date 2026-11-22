<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { HTMLButtonAttributes } from 'svelte/elements';
  import Spinner from './Spinner.svelte';

  type Variant = 'primary' | 'secondary' | 'success' | 'danger' | 'ghost';
  type Size = 'sm' | 'md' | 'lg' | 'pos';

  interface Props extends HTMLButtonAttributes {
    variant?: Variant;
    size?: Size;
    loading?: boolean;
    block?: boolean;
    children: Snippet;
  }

  let {
    variant = 'primary',
    size = 'md',
    loading = false,
    block = false,
    disabled,
    type = 'button',
    children,
    ...rest
  }: Props = $props();
</script>

<button
  {type}
  class="btn {variant} {size}"
  class:block
  disabled={disabled || loading}
  aria-busy={loading}
  {...rest}
>
  {#if loading}<Spinner size={16} />{/if}
  {@render children()}
</button>

<style>
  .btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: var(--space-2);
    border: 1px solid transparent;
    border-radius: var(--radius-md);
    font: inherit;
    font-weight: 600;
    cursor: pointer;
    transition:
      background 0.12s,
      transform 0.06s;
    user-select: none;
  }
  .btn:active:not(:disabled) {
    transform: translateY(1px);
  }
  .btn:disabled {
    opacity: 0.55;
    cursor: not-allowed;
  }
  .block {
    width: 100%;
  }

  .sm {
    padding: var(--space-1) var(--space-3);
    font-size: var(--text-sm);
  }
  .md {
    padding: var(--space-2) var(--space-4);
    min-height: 40px;
  }
  .lg {
    padding: var(--space-3) var(--space-5);
    min-height: var(--touch-min);
    font-size: var(--text-lg);
  }
  .pos {
    flex-direction: column;
    min-height: var(--pos-key-height);
    padding: var(--space-3);
    font-size: var(--text-lg);
  }

  .primary {
    background: var(--color-primary);
    color: var(--color-primary-contrast);
  }
  .primary:hover:not(:disabled) {
    background: var(--color-primary-hover);
  }
  .success {
    background: var(--color-success);
    color: #fff;
  }
  .success:hover:not(:disabled) {
    background: var(--color-success-hover);
  }
  .danger {
    background: var(--color-danger);
    color: #fff;
  }
  .danger:hover:not(:disabled) {
    background: var(--color-danger-hover);
  }
  .secondary {
    background: var(--color-surface);
    color: var(--color-text);
    border-color: var(--color-border);
  }
  .secondary:hover:not(:disabled) {
    background: var(--color-surface-muted);
  }
  .ghost {
    background: transparent;
    color: var(--color-text);
  }
  .ghost:hover:not(:disabled) {
    background: var(--color-surface-muted);
  }
</style>
