<script lang="ts">
  // Name, paper width and characters-per-line of the layout being edited.
  import type { ReceiptTemplateBody } from '@mauzo/types';
  import { TextField } from '@mauzo/ui';

  interface Props {
    name: string;
    body: ReceiptTemplateBody;
    onchange: () => void;
  }
  let { name = $bindable(), body = $bindable(), onchange }: Props = $props();

  function setPaper(p: 58 | 80) {
    body.paper = p;
    delete body.columns;
    onchange();
  }

  function setNarrow(on: boolean) {
    if (on) body.columns = 42;
    else delete body.columns;
    onchange();
  }
</script>

<div class="head">
  <TextField label="Layout name" bind:value={name} maxlength={120} />
  <div role="group" aria-label="Paper width">
    <span class="lbl">Paper</span>
    <div class="seg">
      <button class:on={body.paper === 58} onclick={() => setPaper(58)}>58 mm</button>
      <button class:on={body.paper === 80} onclick={() => setPaper(80)}>80 mm</button>
    </div>
  </div>
</div>
{#if body.paper === 80}
  <label class="chk">
    <input
      type="checkbox"
      checked={body.columns === 42}
      onchange={(e) => setNarrow(e.currentTarget.checked)}
    />
    My printer fits 42 characters per line (not 48)
  </label>
{/if}

<style>
  .head {
    display: grid;
    grid-template-columns: 1fr auto;
    gap: var(--space-3);
    align-items: end;
  }
  .lbl {
    display: block;
    font-size: var(--text-sm);
    font-weight: 600;
    margin-bottom: var(--space-1);
  }
  .seg {
    display: inline-flex;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    overflow: hidden;
  }
  .seg button {
    padding: var(--space-2) var(--space-3);
    background: var(--color-surface);
    border: 0;
    font: inherit;
    font-size: var(--text-sm);
    color: var(--color-text);
    cursor: pointer;
  }
  .seg button + button {
    border-left: 1px solid var(--color-border);
  }
  .seg button.on {
    background: var(--color-primary);
    color: #fff;
  }
  .chk {
    display: flex;
    gap: var(--space-2);
    align-items: center;
    font-size: var(--text-sm);
    margin-top: var(--space-2);
  }
</style>
