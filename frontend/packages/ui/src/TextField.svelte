<script lang="ts">
  import type { HTMLInputAttributes } from 'svelte/elements';

  interface Props extends Omit<HTMLInputAttributes, 'value'> {
    label?: string;
    error?: string | null;
    hint?: string;
    value?: string;
    input?: HTMLInputElement;
  }

  let {
    label,
    error = null,
    hint,
    value = $bindable(''),
    input = $bindable(),
    id = crypto.randomUUID(),
    ...rest
  }: Props = $props();
</script>

<div class="field" class:invalid={!!error}>
  {#if label}<label for={id}>{label}</label>{/if}
  <input
    {id}
    bind:this={input}
    bind:value
    aria-invalid={!!error}
    aria-describedby={error || hint ? `${id}-msg` : undefined}
    {...rest}
  />
  {#if error}
    <p id="{id}-msg" class="msg error">{error}</p>
  {:else if hint}
    <p id="{id}-msg" class="msg">{hint}</p>
  {/if}
</div>

<style>
  .field {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }
  label {
    font-size: var(--text-sm);
    font-weight: 600;
  }
  input {
    min-height: 40px;
    padding: var(--space-2) var(--space-3);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    background: var(--color-surface);
    color: var(--color-text);
    font: inherit;
  }
  .invalid input {
    border-color: var(--color-danger);
  }
  .msg {
    margin: 0;
    font-size: var(--text-xs);
    color: var(--color-text-muted);
  }
  .msg.error {
    color: var(--color-danger);
  }
</style>
