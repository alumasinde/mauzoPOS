<script lang="ts">
  import { Icon } from '@mauzo/ui';
  import { mpesafeed } from '../mpesafeed.svelte';
  // The "More" action sheet — sync, switch cashier, lock, held sales, M-Pesa.
  let {
    onclose,
    onlock,
    onsync,
    onswitch,
    onheld,
    onlookup,
    onmpesa,
  }: {
    onclose: () => void;
    onlock: () => void;
    onsync: () => void;
    onswitch: () => void;
    onheld: () => void;
    onlookup: () => void;
    onmpesa: () => void;
  } = $props();

  function run(fn: () => void) {
    return () => {
      fn();
      onclose();
    };
  }
</script>

<div class="modal">
  <h3>More</h3>
  <div class="grid">
    <button class="btn" onclick={run(onsync)}><Icon name="refresh" size={26} /><span>Sync now</span></button>
    <button class="btn" onclick={run(onswitch)}>
      <Icon name="user" size={26} /><span>Switch cashier</span>
    </button>
    <button class="btn" onclick={run(onlock)}><Icon name="lock" size={26} /><span>Lock till</span></button>
    <button class="btn" onclick={onheld}><Icon name="pause" size={26} /><span>Held sales</span></button>
    <button class="btn" onclick={onlookup}><Icon name="search" size={26} /><span>Find sale</span></button>
    <button class="btn" onclick={onmpesa}>
      <Icon name="phone" size={26} /><span>M-Pesa</span>
      {#if mpesafeed.count > 0}<em class="badge">{mpesafeed.count}</em>{/if}
    </button>
  </div>
  <button class="ghost" onclick={onclose}>Close</button>
</div>

<style>
  .modal {
    width: min(420px, 92vw);
    background: var(--color-surface);
    border-radius: var(--radius-lg);
    padding: var(--space-5);
    box-shadow: 0 12px 40px rgb(0 0 0 / 0.3);
  }
  h3 {
    margin: 0 0 var(--space-3);
  }
  .grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--space-2);
    margin-bottom: var(--space-3);
  }
  .btn {
    position: relative;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-1);
    padding: var(--space-4);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    background: var(--color-surface);
    cursor: pointer;
    color: var(--color-primary);
  }
  .badge {
    position: absolute;
    top: 6px;
    right: 10px;
    background: #00a651;
    color: #fff;
    font-size: 0.7rem;
    font-style: normal;
    font-weight: 700;
    border-radius: 999px;
    padding: 1px 7px;
  }
  .btn span {
    font-size: var(--text-sm);
    font-weight: 600;
  }
  .ghost {
    margin-top: var(--space-3);
    padding: var(--space-2) var(--space-4);
    font: inherit;
    background: none;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    cursor: pointer;
  }
</style>
