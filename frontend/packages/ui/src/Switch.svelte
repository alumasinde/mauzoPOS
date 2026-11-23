<script lang="ts">
  interface Props {
    checked: boolean;
    label: string;
    disabled?: boolean;
    busy?: boolean;
    onchange?: (next: boolean) => void;
  }

  let { checked, label, disabled = false, busy = false, onchange }: Props = $props();
</script>

<button
  type="button"
  role="switch"
  class="switch"
  class:on={checked}
  aria-checked={checked}
  aria-label={label}
  aria-busy={busy}
  disabled={disabled || busy}
  onclick={() => onchange?.(!checked)}
>
  <span class="knob"></span>
</button>

<style>
  .switch {
    position: relative;
    width: 44px;
    height: 24px;
    flex: none;
    border: none;
    border-radius: 999px;
    background: var(--color-border);
    cursor: pointer;
    transition: background 0.15s;
  }
  .switch.on {
    background: var(--color-success);
  }
  .switch:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
  .knob {
    position: absolute;
    top: 3px;
    left: 3px;
    width: 18px;
    height: 18px;
    border-radius: 50%;
    background: #fff;
    box-shadow: var(--shadow-sm);
    transition: transform 0.15s;
  }
  .on .knob {
    transform: translateX(20px);
  }
</style>
