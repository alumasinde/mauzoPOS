<script lang="ts">
  import { onMount } from 'svelte';
  import { desktop, type Cashier, type SyncResult } from '../desktop';

  let { onpick }: { onpick: (c: Cashier) => void } = $props();

  const bridge = desktop();
  let cashiers = $state<Cashier[]>([]);
  let syncing = $state(false);
  let syncMsg = $state('');
  let error = $state('');

  async function load() {
    if (!bridge) return;
    try {
      cashiers = await bridge.Cashiers();
    } catch (e) {
      error = e instanceof Error ? e.message : 'Could not load cashiers';
    }
  }

  async function sync() {
    if (!bridge) return;
    syncing = true;
    error = '';
    try {
      const r: SyncResult = await bridge.Sync();
      if (r.ok) {
        syncMsg = `Synced ${r.products} products · ${r.cashiers} cashiers`;
        await load();
      } else {
        error = r.error ?? 'Sync failed';
      }
    } finally {
      syncing = false;
    }
  }

  onMount(async () => {
    await load();
    if (cashiers.length === 0) await sync(); // first run: pull the working set
  });
</script>

<div class="screen">
  <div class="card">
    <h1>Who's at the till?</h1>
    {#if cashiers.length}
      <div class="grid">
        {#each cashiers as c (c.id)}
          <button class="who" onclick={() => onpick(c)}>
            <span class="avatar">
              {c.name
                .split(' ')
                .map((w) => w[0])
                .slice(0, 2)
                .join('')}
            </span>
            <span class="nm">{c.name}</span>
            <span class="role">{c.role_name}</span>
          </button>
        {/each}
      </div>
    {:else}
      <p class="muted">No cashiers synced yet. Pull the catalog from the server.</p>
    {/if}
    {#if syncMsg}<p class="okmsg">{syncMsg}</p>{/if}
    {#if error}<p class="err">{error}</p>{/if}
    <button class="sync" onclick={sync} disabled={syncing}>
      {syncing ? 'Syncing…' : 'Sync from server'}
    </button>
  </div>
</div>

<style>
  .screen {
    display: grid;
    place-items: center;
    height: 100vh;
    background: var(--color-surface-muted);
  }
  .card {
    width: min(720px, 94vw);
    padding: var(--space-6);
    background: var(--color-surface);
    border-radius: var(--radius-lg);
  }
  h1 {
    margin: 0 0 var(--space-4);
  }
  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
    gap: var(--space-3);
  }
  .who {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-4);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    background: var(--color-surface);
    cursor: pointer;
  }
  .who:hover {
    border-color: var(--color-primary);
    background: var(--color-surface-muted);
  }
  .avatar {
    width: 52px;
    height: 52px;
    border-radius: 50%;
    display: grid;
    place-items: center;
    font-weight: 700;
    background: var(--color-primary);
    color: #fff;
    font-size: var(--text-lg);
  }
  .nm {
    font-weight: 600;
  }
  .role {
    font-size: var(--text-xs);
    color: var(--color-text-muted);
    text-transform: uppercase;
  }
  .muted {
    color: var(--color-text-muted);
  }
  .err {
    color: var(--color-danger);
  }
  .okmsg {
    color: var(--color-success);
  }
  .sync {
    margin-top: var(--space-4);
    padding: var(--space-2) var(--space-4);
    font: inherit;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    background: none;
    cursor: pointer;
  }
</style>
