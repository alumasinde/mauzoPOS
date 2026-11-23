<script lang="ts">
  // Label … value rows of a "Details" block.
  import type { ReceiptField } from '@mauzo/types';
  import { Button, Icon } from '@mauzo/ui';

  interface Props {
    fields: ReceiptField[];
    /** The row/part last focused — where "Insert a variable" goes. */
    focused: { i: number; part: 'label' | 'value' } | null;
    onchange: () => void;
  }
  let { fields = $bindable(), focused = $bindable(), onchange }: Props = $props();

  function add() {
    fields = [...fields, { label: 'Label', value: '' }];
    onchange();
  }
  function remove(i: number) {
    fields = fields.filter((_, j) => j !== i);
    onchange();
  }
</script>

<span class="lbl">Rows</span>
{#each fields as f, i (i)}
  <div class="frow">
    <input
      aria-label="Label {i + 1}"
      value={f.label}
      onfocus={() => (focused = { i, part: 'label' })}
      oninput={(e) => {
        f.label = e.currentTarget.value;
        onchange();
      }}
    />
    <input
      aria-label="Value {i + 1}"
      value={f.value}
      placeholder={'{{sale.cashier}}'}
      onfocus={() => (focused = { i, part: 'value' })}
      oninput={(e) => {
        f.value = e.currentTarget.value;
        onchange();
      }}
    />
    <button class="icon" aria-label="Remove row {i + 1}" onclick={() => remove(i)}>
      <Icon name="x" size={16} />
    </button>
  </div>
{/each}
<Button size="sm" variant="secondary" onclick={add}>Add row</Button>
<p class="hint">A row is skipped when its value is empty (e.g. no customer).</p>

<style>
  .lbl {
    font-size: var(--text-sm);
    font-weight: 600;
    margin-top: var(--space-2);
  }
  .frow {
    display: grid;
    grid-template-columns: 1fr 1.4fr auto;
    gap: var(--space-1);
  }
  input {
    width: 100%;
    padding: var(--space-2);
    font: inherit;
    font-size: var(--text-sm);
    color: var(--color-text);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
  }
  .icon {
    display: grid;
    place-items: center;
    background: none;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    color: var(--color-text-muted);
    cursor: pointer;
    padding: 0 var(--space-2);
  }
  .hint {
    margin: 0;
    font-size: var(--text-xs);
    color: var(--color-text-muted);
  }
</style>
