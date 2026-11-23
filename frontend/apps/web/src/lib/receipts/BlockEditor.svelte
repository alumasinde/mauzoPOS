<script lang="ts">
  // Edits one layout block in place (the parent owns the template body).
  import type { NamedDoc, ReceiptBlock } from '@mauzo/types';
  import { Icon } from '@mauzo/ui';
  import { ALIGNED, STYLED, kindOf } from './blocks';
  import ConditionPicker from './ConditionPicker.svelte';
  import FieldRows from './FieldRows.svelte';

  interface Props {
    block: ReceiptBlock;
    variables: NamedDoc[];
    conditions: NamedDoc[];
    onchange: () => void;
  }
  let { block = $bindable(), variables, conditions, onchange }: Props = $props();

  let textArea = $state<HTMLTextAreaElement | null>(null);
  let focusedField = $state<{ i: number; part: 'label' | 'value' } | null>(null);
  let pickVar = $state('');

  const kind = $derived(kindOf(block.type));

  function set<K extends keyof ReceiptBlock>(key: K, value: ReceiptBlock[K]) {
    if (value === '' || value === undefined || value === false) delete block[key];
    else block[key] = value;
    onchange();
  }

  // Insert a {{variable}} where the cursor is (text) or into the last
  // focused field value.
  function insertVar(name: string) {
    if (!name) return;
    const token = `{{${name}}}`;
    if (block.type === 'text') {
      const el = textArea;
      const cur = block.text ?? '';
      const at = el?.selectionStart ?? cur.length;
      block.text = cur.slice(0, at) + token + cur.slice(el?.selectionEnd ?? at);
    } else if (block.type === 'fields' && block.fields?.length) {
      const f = focusedField ?? { i: block.fields.length - 1, part: 'value' as const };
      const row = block.fields[f.i];
      if (row) row[f.part] = (row[f.part] ?? '') + token;
    } else if (block.type === 'qr' || block.type === 'barcode') {
      block.data = (block.data ?? '') + token;
    }
    pickVar = '';
    onchange();
  }

  const takesVars = $derived(['text', 'fields', 'qr', 'barcode'].includes(block.type));
</script>

<div class="editor">
  <header>
    <Icon name={kind.icon} size={18} />
    <strong>{kind.label}</strong>
  </header>
  <p class="hint">{kind.hint}</p>

  {#if block.type === 'text'}
    <label class="lbl" for="blk-text">Text</label>
    <textarea
      id="blk-text"
      rows="3"
      bind:this={textArea}
      value={block.text ?? ''}
      oninput={(e) => set('text', e.currentTarget.value)}></textarea>
    <p class="hint">Press Enter for a new line. A line whose variables are all empty is not printed.</p>
  {:else if block.type === 'fields'}
    <FieldRows
      bind:fields={() => block.fields ?? [], (v) => (block.fields = v)}
      bind:focused={focusedField}
      {onchange}
    />
  {:else if block.type === 'divider'}
    <label class="lbl" for="blk-char">Character</label>
    <select
      id="blk-char"
      value={block.char ?? '-'}
      onchange={(e) => set('char', e.currentTarget.value === '-' ? undefined : e.currentTarget.value)}
    >
      <option value="-">- - - -</option>
      <option value="=">= = = =</option>
      <option value="*">* * * *</option>
      <option value=".">. . . .</option>
      <option value="_">_ _ _ _</option>
    </select>
  {:else if block.type === 'items'}
    <label class="lbl" for="blk-layout">Layout</label>
    <select
      id="blk-layout"
      value={block.layout ?? 'detailed'}
      onchange={(e) => set('layout', e.currentTarget.value as 'detailed' | 'compact')}
    >
      <option value="detailed">Name, then quantity × price (easier to read)</option>
      <option value="compact">One line per item (shorter receipts)</option>
    </select>
    <label class="chk">
      <input
        type="checkbox"
        checked={!!block.show_sku}
        onchange={(e) => set('show_sku', e.currentTarget.checked)}
      /> Show SKU
    </label>
    <label class="chk">
      <input
        type="checkbox"
        checked={!!block.show_promo}
        onchange={(e) => set('show_promo', e.currentTarget.checked)}
      /> Show promotion names
    </label>
  {:else if block.type === 'totals'}
    <label class="chk">
      <input
        type="checkbox"
        checked={!!block.show_item_count}
        onchange={(e) => set('show_item_count', e.currentTarget.checked)}
      /> Show number of items
    </label>
  {:else if block.type === 'qr' || block.type === 'barcode'}
    <label class="lbl" for="blk-data">{block.type === 'qr' ? 'QR content' : 'Barcode content'}</label>
    <input id="blk-data" value={block.data ?? ''} oninput={(e) => set('data', e.currentTarget.value)} />
    {#if block.type === 'qr'}
      <label class="lbl" for="blk-qrsize">Size</label>
      <select
        id="blk-qrsize"
        value={String(block.qr_size ?? 6)}
        onchange={(e) => set('qr_size', Number(e.currentTarget.value))}
      >
        {#each [3, 4, 5, 6, 7, 8, 10] as n (n)}<option value={String(n)}>{n}</option>{/each}
      </select>
    {/if}
  {:else if block.type === 'feed'}
    <label class="lbl" for="blk-lines">Blank lines</label>
    <input
      id="blk-lines"
      type="number"
      min="1"
      max="10"
      value={block.lines ?? 1}
      oninput={(e) => set('lines', Math.min(10, Math.max(1, Number(e.currentTarget.value) || 1)))}
    />
  {/if}

  {#if ALIGNED.includes(block.type)}
    <span class="lbl">Alignment</span>
    <div class="seg" role="group" aria-label="Alignment">
      {#each ['left', 'center', 'right'] as a (a)}
        <button
          class:on={(block.align ?? (block.type === 'text' ? 'left' : 'center')) === a}
          onclick={() => set('align', a as 'left' | 'center' | 'right')}
        >
          {a}
        </button>
      {/each}
    </div>
  {/if}
  {#if STYLED.includes(block.type)}
    <span class="lbl">Style</span>
    <div class="row">
      <label class="chk">
        <input
          type="checkbox"
          checked={!!block.bold}
          onchange={(e) => set('bold', e.currentTarget.checked)}
        /> Bold
      </label>
      <select
        aria-label="Size"
        value={block.size ?? 'normal'}
        onchange={(e) =>
          set(
            'size',
            e.currentTarget.value === 'normal'
              ? undefined
              : (e.currentTarget.value as 'tall' | 'wide' | 'double'),
          )}
      >
        <option value="normal">Normal size</option>
        <option value="tall">Tall</option>
        <option value="wide">Wide</option>
        <option value="double">Double (headings)</option>
      </select>
    </div>
  {/if}

  {#if takesVars}
    <label class="lbl" for="blk-var">Insert a variable</label>
    <select id="blk-var" bind:value={pickVar} onchange={() => insertVar(pickVar)}>
      <option value="">Choose…</option>
      {#each variables as v (v.name)}<option value={v.name}>
          {v.description} — {`{{${v.name}}}`}
        </option>{/each}
    </select>
  {/if}

  <ConditionPicker value={block.if} {variables} {conditions} onset={(v) => set('if', v)} />
</div>

<style>
  .editor {
    display: grid;
    gap: var(--space-2);
  }
  header {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    color: var(--color-primary);
  }
  header strong {
    color: var(--color-text);
  }
  .hint {
    margin: 0;
    font-size: var(--text-xs);
    color: var(--color-text-muted);
  }
  .lbl {
    font-size: var(--text-sm);
    font-weight: 600;
    margin-top: var(--space-2);
  }
  textarea,
  input:not([type='checkbox']),
  select {
    width: 100%;
    padding: var(--space-2);
    font: inherit;
    font-size: var(--text-sm);
    color: var(--color-text);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
  }
  textarea {
    font-family: var(--font-mono);
    resize: vertical;
  }
  .chk {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    font-size: var(--text-sm);
    white-space: nowrap;
  }
  .row {
    display: flex;
    gap: var(--space-3);
    align-items: center;
  }
  .seg {
    display: inline-flex;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    overflow: hidden;
    width: fit-content;
  }
  .seg button {
    padding: var(--space-1) var(--space-3);
    background: var(--color-surface);
    border: 0;
    font: inherit;
    font-size: var(--text-sm);
    text-transform: capitalize;
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
</style>
