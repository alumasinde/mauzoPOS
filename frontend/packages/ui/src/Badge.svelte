<script lang="ts">
  import type { Snippet } from 'svelte';

  type Tone = 'neutral' | 'info' | 'success' | 'warning' | 'danger';
  let { tone = 'neutral', title, children }: { tone?: Tone; title?: string; children: Snippet } = $props();
</script>

<span class="badge {tone}" {title}>{@render children()}</span>

<style>
  .badge {
    display: inline-flex;
    align-items: center;
    gap: var(--space-1);
    padding: 2px var(--space-2);
    border-radius: 999px;
    font-size: var(--text-xs);
    font-weight: 600;
    white-space: nowrap;
    background: var(--color-surface-muted);
    color: var(--color-text-muted);
    border: 1px solid var(--color-border);
  }
  .info {
    color: var(--color-primary);
    border-color: color-mix(in srgb, var(--color-primary) 35%, transparent);
  }
  .success {
    color: var(--color-success);
    border-color: color-mix(in srgb, var(--color-success) 35%, transparent);
  }
  .warning {
    color: #9a6700;
    border-color: color-mix(in srgb, var(--color-warning) 50%, transparent);
  }
  .danger {
    color: var(--color-danger);
    border-color: color-mix(in srgb, var(--color-danger) 35%, transparent);
  }
</style>
