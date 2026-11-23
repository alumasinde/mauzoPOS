<script lang="ts">
  // Library + block editor + live preview.
  import { Button, Icon } from '@mauzo/ui';
  import BlockEditor from './BlockEditor.svelte';
  import type { Designer } from './designer.svelte.ts';
  import HistoryModal from './HistoryModal.svelte';
  import LayoutHeader from './LayoutHeader.svelte';
  import LayoutLibrary from './LayoutLibrary.svelte';
  import LayoutList from './LayoutList.svelte';
  import NewLayoutModal from './NewLayoutModal.svelte';
  import PreviewPane from './PreviewPane.svelte';

  let { d }: { d: Designer } = $props();

  let newOpen = $state(false);
  let historyOpen = $state(false);

  function startNew() {
    if (d.canLeave()) newOpen = true;
  }

  async function showHistory() {
    historyOpen = await d.loadHistory();
  }
</script>

<div class="designer">
  <aside class="card">
    <LayoutLibrary
      templates={d.templates}
      currentId={d.currentId}
      onopen={(t) => d.open(t)}
      onnew={startNew}
    />
  </aside>
  {#if d.body}
    <section class="card">
      <LayoutHeader bind:name={d.name} bind:body={d.body} onchange={() => d.schedulePreview()} />
      <div class="cols">
        <div>
          <h3>Blocks</h3>
          <LayoutList
            bind:blocks={d.body.blocks}
            selected={d.selected}
            onselect={(i) => (d.selected = i)}
            onchange={() => d.schedulePreview()}
          />
        </div>
        <div class="props">
          {#if d.body.blocks[d.selected] && d.meta}
            <BlockEditor
              bind:block={
                () => d.body!.blocks[d.selected]!,
                (v) => {
                  d.body!.blocks[d.selected] = v;
                }
              }
              variables={d.meta.variables}
              conditions={d.meta.conditions}
              onchange={() => d.schedulePreview()}
            />
          {:else}
            <p class="muted">Select a block to edit it.</p>
          {/if}
        </div>
      </div>
      <div class="actions">
        <Button onclick={() => d.save()} loading={d.saving} disabled={!d.dirty || d.problems.length > 0}>
          {d.dirty ? 'Save layout' : 'Saved'}
        </Button>
        <Button variant="secondary" onclick={showHistory}>History</Button>
        <Button variant="ghost" onclick={() => d.archive()}>Delete</Button>
        {#if d.dirty}<span class="unsaved">Unsaved changes</span>{/if}
      </div>
    </section>
    <section class="card preview">
      <PreviewPane
        output={d.preview}
        problems={d.problems}
        logoUrl={d.logoUrl}
        recent={d.recent}
        source={d.source}
        onsource={(id) => d.setSource(id)}
      />
    </section>
  {:else}
    <section class="card empty">
      <Icon name="printer" size={36} />
      <p>
        Create a layout to start designing. Until you do, every till prints the default for your business
        type.
      </p>
      <Button onclick={startNew}>Create a layout</Button>
    </section>
  {/if}
</div>

<NewLayoutModal
  bind:open={newOpen}
  builtins={d.meta?.builtins ?? []}
  templates={d.templates}
  oncreated={(t) => d.created(t)}
/>
<HistoryModal
  bind:open={historyOpen}
  versions={d.versions}
  current={d.version}
  onrestore={(v) => {
    historyOpen = false;
    d.restore(v);
  }}
/>

<style>
  .designer {
    display: grid;
    grid-template-columns: 220px minmax(0, 1.25fr) minmax(0, 1fr);
    gap: var(--space-4);
    align-items: start;
  }
  .card {
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-lg);
    padding: var(--space-4) var(--space-5);
    box-shadow: var(--shadow-sm);
    min-width: 0;
  }
  h3 {
    margin: 0 0 var(--space-2);
    font-size: var(--text-sm);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--color-text-muted);
  }
  .muted {
    color: var(--color-text-muted);
    margin: 0;
  }
  .cols {
    display: grid;
    grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
    gap: var(--space-4);
    margin-top: var(--space-4);
  }
  .props {
    border-left: 1px solid var(--color-border);
    padding-left: var(--space-4);
  }
  .actions {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    margin-top: var(--space-4);
    padding-top: var(--space-3);
    border-top: 1px solid var(--color-border);
  }
  .unsaved {
    font-size: var(--text-sm);
    color: var(--color-warning);
    font-weight: 600;
  }
  .preview {
    position: sticky;
    top: 84px;
  }
  .empty {
    grid-column: span 2;
    display: grid;
    justify-items: center;
    gap: var(--space-3);
    padding: var(--space-6);
    text-align: center;
    color: var(--color-text-muted);
  }
  @media (max-width: 1280px) {
    .designer {
      grid-template-columns: 200px minmax(0, 1fr);
    }
    .preview {
      grid-column: 2;
      position: static;
    }
  }
  @media (max-width: 860px) {
    .designer,
    .cols {
      grid-template-columns: minmax(0, 1fr);
    }
    .preview {
      grid-column: auto;
    }
    .props {
      border-left: 0;
      padding-left: 0;
    }
  }
</style>
