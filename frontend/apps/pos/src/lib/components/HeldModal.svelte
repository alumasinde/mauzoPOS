<script lang="ts">
  // Held-sales modal — park the current cart, recall or drop parked sales.
  import { desktop, type Held } from '../desktop';
  import { cart } from '../cart.svelte';

  let {
    onclose,
    onhold,
  }: {
    onclose: () => void;
    onhold: (label: string) => void;
  } = $props();

  const bridge = desktop();
  let list = $state<Held[]>([]);
  let label = $state('');

  $effect(() => {
    void load();
  });

  async function load() {
    if (!bridge) return;
    list = await bridge.HeldList().catch(() => []);
  }

  async function recall(id: string) {
    if (!bridge) return;
    const h = await bridge.RecallHeld(id).catch(() => null);
    if (h) {
      await cart.restore(h.cart);
      onclose();
    }
  }

  async function drop(id: string) {
    if (!bridge) return;
    await bridge.CancelHeld(id).catch(() => {});
    await load();
  }

  function hold() {
    onhold(label);
    onclose();
  }
</script>

<div class="overlay" role="presentation" onclick={(e) => e.target === e.currentTarget && onclose()}>
  <div class="modal">
    <h3>Held sales</h3>
    <div class="holdRow">
      <input bind:value={label} placeholder="Label this sale (e.g. table 4, John)" />
      <button class="go" onclick={hold} disabled={cart.count === 0}>Hold current</button>
    </div>
    <div class="heldList">
      {#each list as h (h.id)}
        <div class="held">
          <div>
            <b>{h.label || 'Held sale'}</b>
            <small>{h.held_by} · {h.held_at}{h.source === 'local' ? ' · this till' : ''}</small>
          </div>
          <button class="go" onclick={() => recall(h.id)}>Recall</button>
          <button class="ghost sm" onclick={() => drop(h.id)}>Delete</button>
        </div>
      {:else}
        <p class="muted">Nothing parked.</p>
      {/each}
    </div>
    <button class="ghost" onclick={onclose}>Close</button>
  </div>
</div>

<style>
  .overlay {
    position: fixed;
    inset: 0;
    background: rgb(4 10 24 / 0.55);
    display: grid;
    place-items: center;
    z-index: 40;
  }
  .modal {
    width: min(560px, 94vw);
    background: var(--color-surface);
    border-radius: var(--radius-lg);
    padding: var(--space-5);
    box-shadow: 0 12px 40px rgb(0 0 0 / 0.3);
  }
  h3 {
    margin: 0 0 var(--space-3);
  }
  .muted {
    color: var(--color-text-muted);
  }
  input {
    width: 100%;
    padding: var(--space-3);
    font: inherit;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    box-sizing: border-box;
  }
  .holdRow {
    display: flex;
    gap: var(--space-2);
    margin-bottom: var(--space-3);
  }
  .go {
    padding: var(--space-3) var(--space-4);
    font-weight: 700;
    color: #fff;
    background: var(--color-primary);
    border: none;
    border-radius: var(--radius-md);
    cursor: pointer;
    white-space: nowrap;
  }
  .go:disabled {
    opacity: 0.5;
  }
  .ghost {
    padding: var(--space-2) var(--space-4);
    font: inherit;
    background: none;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    cursor: pointer;
  }
  .ghost.sm {
    padding: var(--space-2);
  }
  .heldList {
    max-height: 280px;
    overflow-y: auto;
    margin-bottom: var(--space-3);
  }
  .held {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-2) 0;
    border-bottom: 1px solid var(--color-surface-muted);
  }
  .held > div {
    flex: 1;
  }
  .held small {
    display: block;
    color: var(--color-text-muted);
  }
</style>
