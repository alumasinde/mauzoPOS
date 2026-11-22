<script lang="ts">
  import { toasts, dismissToast } from './toast.svelte';
</script>

<div class="toaster" aria-live="polite">
  {#each toasts as t (t.id)}
    <div class="toast {t.kind}" role={t.kind === 'error' ? 'alert' : 'status'}>
      <span>{t.message}</span>
      <button aria-label="Dismiss" onclick={() => dismissToast(t.id)}>×</button>
    </div>
  {/each}
</div>

<style>
  .toaster {
    position: fixed;
    right: var(--space-4);
    bottom: var(--space-4);
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    z-index: 1000;
  }
  .toast {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    min-width: 260px;
    max-width: 420px;
    padding: var(--space-3) var(--space-4);
    border-radius: var(--radius-md);
    color: #fff;
    box-shadow: var(--shadow-md);
  }
  .success {
    background: var(--color-success);
  }
  .error {
    background: var(--color-danger);
  }
  .info {
    background: var(--color-primary);
  }
  .warning {
    background: var(--color-warning);
    color: #1a1a1a;
  }
  span {
    flex: 1;
  }
  button {
    border: none;
    background: none;
    color: inherit;
    font-size: var(--text-lg);
    cursor: pointer;
  }
</style>
