<script lang="ts">
  type Status = 'ok' | 'warn' | 'error' | 'unknown';
  let { status, label }: { status: Status; label: string } = $props();
</script>

<span class="status {status}"><span class="dot" aria-hidden="true"></span>{label}</span>

<style>
  .status {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    font-weight: 600;
  }
  .dot {
    width: 10px;
    height: 10px;
    border-radius: 50%;
    background: var(--color-neutral);
  }
  .ok .dot {
    background: var(--color-success);
  }
  .warn .dot {
    background: var(--color-warning);
  }
  .error .dot {
    background: var(--color-danger);
  }
</style>
