<script lang="ts">
  // Customer picker — search synced accounts and attach one to the cart
  // (required for credit sales, optional otherwise).
  import { desktop, type Customer } from '../desktop';
  import { cart } from '../cart.svelte';
  import { formatMoney } from '../money';

  const bridge = desktop();
  let q = $state('');
  let custs = $state<Customer[]>([]);
  let busy = $state(false);
  let open = $state(false);

  async function find() {
    if (!bridge) return;
    busy = true;
    try {
      custs = await bridge.Customers(q.trim());
      open = true;
    } catch {
      custs = [];
    } finally {
      busy = false;
    }
  }

  function pick(c: Customer) {
    cart.setCustomer(c.id, c.name);
    open = false;
    q = '';
  }
</script>

<div class="cust">
  {#if cart.customerID}
    <span class="cpick">
      Customer: <b>{cart.customerName}</b>
      <button class="x" onclick={() => cart.setCustomer('', '')} aria-label="Clear">✕</button>
    </span>
  {:else}
    <div class="pcRow">
      <input
        bind:value={q}
        placeholder="Customer name or phone…"
        onkeydown={(e) => e.key === 'Enter' && find()}
      />
      <button class="go sm" onclick={find} disabled={busy}>{busy ? '…' : 'Find'}</button>
    </div>
    {#if open}
      <ul class="clist">
        {#each custs as c (c.id)}
          <li>
            <button onclick={() => pick(c)}>
              <b>{c.name}</b> <span class="dim">{c.phone || c.code || ''}</span>
              {#if c.credit_limit}<span class="dim">limit {formatMoney(c.credit_limit)}</span>{/if}
            </button>
          </li>
        {:else}<li class="dim">No customers found.</li>{/each}
      </ul>
    {/if}
  {/if}
</div>

<style>
  .cust {
    margin-bottom: var(--space-3);
  }
  .pcRow {
    display: flex;
    gap: var(--space-2);
  }
  .pcRow input {
    flex: 1;
    padding: 10px;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    font: inherit;
  }
  .go.sm {
    flex: none;
    padding: 8px 14px;
    border: 0;
    background: var(--color-primary);
    color: #fff;
    border-radius: var(--radius-md);
    cursor: pointer;
    font-weight: 600;
  }
  .clist {
    list-style: none;
    margin: var(--space-2) 0 0;
    padding: 0;
    max-height: 160px;
    overflow: auto;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
  }
  .clist li button {
    display: block;
    width: 100%;
    text-align: left;
    padding: 8px 10px;
    border: 0;
    background: none;
    cursor: pointer;
    font: inherit;
    border-bottom: 1px solid var(--color-border);
  }
  .clist li button:hover {
    background: var(--color-surface-muted);
  }
  .clist li.dim {
    padding: 8px 10px;
    color: var(--color-text-muted);
  }
  .cpick {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    font-size: var(--text-sm);
  }
  .x {
    border: 0;
    background: none;
    color: var(--color-danger);
    cursor: pointer;
    font-size: 14px;
  }
  .dim {
    color: var(--color-text-muted);
    font-size: var(--text-xs);
  }
</style>
