<script lang="ts">
  import { Icon } from '@mauzo/ui';
  // Incoming M-Pesa collections — the live feed of confirmed payments waiting
  // to fund a sale. "Use" drops the code into the pay modal; "Allocate" links
  // it to a completed receipt at this branch.
  import { desktop, type MpesaTxn } from '../desktop';
  import { mpesafeed } from '../mpesafeed.svelte';
  import { cart } from '../cart.svelte';
  import { formatMoney } from '../money';

  let {
    onclose,
    onuse,
  }: {
    onclose: () => void;
    onuse: () => void; // open the pay modal (mpesa method)
  } = $props();

  const bridge = desktop();
  let allocFor = $state('');
  let saleRef = $state('');
  let busy = $state(false);
  let error = $state('');

  function when(t: MpesaTxn): string {
    const d = t.trans_time || t.created_at;
    if (!d) return '';
    return new Date(d).toLocaleTimeString('en-KE', { hour: '2-digit', minute: '2-digit' });
  }

  function use(t: MpesaTxn) {
    cart.mpesaRef = t.trans_id ?? '';
    onuse();
  }

  async function allocate(t: MpesaTxn) {
    if (!bridge || !saleRef.trim()) return;
    busy = true;
    error = '';
    try {
      await bridge.MpesaAllocate(t.public_id, saleRef.trim());
      mpesafeed.drop(t.public_id);
      allocFor = '';
      saleRef = '';
    } catch (e) {
      error = e instanceof Error ? e.message : 'Allocation failed';
    } finally {
      busy = false;
    }
  }
</script>

<div class="modal">
  <h3>M-Pesa payments <span class="count">{mpesafeed.count}</span></h3>
  <p class="muted">Confirmed collections not yet linked to a sale.</p>

  {#if mpesafeed.items.length === 0}
    <p class="empty">Nothing waiting — payments land here the moment they confirm.</p>
  {:else}
    <div class="list">
      {#each mpesafeed.items as t (t.public_id)}
        <div class="item">
          <div class="lead">
            <span class="amt">KSh {formatMoney(t.amount)}</span>
            <code class="code">{t.trans_id}</code>
          </div>
          <div class="meta">
            <span>{t.payer_name || t.phone}</span>
            <span>{when(t)}</span>
          </div>
          {#if allocFor === t.public_id}
            <div class="alloc">
              <input bind:value={saleRef} inputmode="numeric" placeholder="Receipt #" />
              <button class="go" onclick={() => allocate(t)} disabled={busy || !saleRef.trim()}>
                {busy ? '…' : 'Link'}
              </button>
              <button class="x" onclick={() => (allocFor = '')} aria-label="Cancel">
                <Icon name="x" size={13} />
              </button>
            </div>
          {:else}
            <div class="acts">
              <button class="use" onclick={() => use(t)}>Use in payment</button>
              <button
                class="ghost"
                onclick={() => {
                  allocFor = t.public_id;
                  saleRef = '';
                  error = '';
                }}
              >
                Allocate…
              </button>
            </div>
          {/if}
        </div>
      {/each}
    </div>
  {/if}

  {#if error}<p class="err">{error}</p>{/if}
  <button class="close" onclick={onclose}>Close</button>
</div>

<style>
  .modal {
    width: min(460px, 94vw);
    max-height: 80vh;
    display: flex;
    flex-direction: column;
    background: var(--color-surface);
    border-radius: var(--radius-lg);
    padding: var(--space-5);
    box-shadow: 0 12px 40px rgb(0 0 0 / 0.3);
  }
  h3 {
    margin: 0 0 var(--space-1);
  }
  .count {
    font-size: var(--text-sm);
    background: #00a651;
    color: #fff;
    border-radius: 999px;
    padding: 2px 9px;
    vertical-align: middle;
  }
  .muted {
    color: var(--color-text-muted);
    font-size: var(--text-sm);
    margin: 0 0 var(--space-3);
  }
  .empty {
    color: var(--color-text-muted);
    text-align: center;
    padding: var(--space-5) 0;
  }
  .list {
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }
  .item {
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    padding: var(--space-3);
  }
  .lead {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
  }
  .amt {
    font-weight: 800;
    font-size: var(--text-lg);
  }
  .code {
    font-size: var(--text-sm);
    color: var(--color-primary);
    font-weight: 700;
  }
  .meta {
    display: flex;
    justify-content: space-between;
    color: var(--color-text-muted);
    font-size: var(--text-sm);
    margin: 2px 0 var(--space-2);
  }
  .acts {
    display: flex;
    gap: var(--space-2);
  }
  .use {
    flex: 1;
    padding: 8px;
    border: 0;
    border-radius: var(--radius-md);
    background: #00a651;
    color: #fff;
    font-weight: 700;
    cursor: pointer;
  }
  .ghost {
    padding: 8px 12px;
    border: 1px solid var(--color-border);
    background: none;
    border-radius: var(--radius-md);
    cursor: pointer;
    font: inherit;
    font-size: var(--text-sm);
  }
  .alloc {
    display: flex;
    gap: var(--space-2);
  }
  .alloc input {
    flex: 1;
    padding: 8px;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    font: inherit;
  }
  .alloc .go {
    padding: 8px 14px;
    border: 0;
    border-radius: var(--radius-md);
    background: var(--color-primary);
    color: #fff;
    font-weight: 700;
    cursor: pointer;
  }
  .alloc .x {
    border: 0;
    background: none;
    color: var(--color-danger);
    cursor: pointer;
  }
  .err {
    color: var(--color-danger);
    font-size: var(--text-sm);
    margin: var(--space-2) 0 0;
  }
  .close {
    margin-top: var(--space-3);
    padding: 10px;
    border: 1px solid var(--color-border);
    background: none;
    border-radius: var(--radius-md);
    font: inherit;
    cursor: pointer;
  }
</style>
