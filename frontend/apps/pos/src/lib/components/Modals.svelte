<script lang="ts">
  // Checkout modals: price check, quantity, discount, supervisor PIN, the
  // "more" menu, and the unknown-barcode prompt. Pay and held-sale modals are
  // their own components; open() routes to them.
  import { desktop, type PayResult, type PriceCheckResult } from '../desktop';
  import { cart } from '../cart.svelte';
  import { formatMoney } from '../money';
  import PinPad from './PinPad.svelte';
  import PayModal from './PayModal.svelte';
  import HeldModal from './HeldModal.svelte';
  import ValuePrompt from './ValuePrompt.svelte';
  import MoreModal from './MoreModal.svelte';
  import LookupModal from './LookupModal.svelte';

  type Modal =
    | 'none'
    | 'pricecheck'
    | 'quantity'
    | 'discount'
    | 'held'
    | 'supervisor'
    | 'pay'
    | 'more'
    | 'unknown'
    | 'lookup';

  let {
    currency = 'KSh',
    method = 'cash',
    onpaid,
    onhold,
    onlock,
    onsync,
    onswitch,
  }: {
    currency?: string;
    method?: string;
    onpaid: (r: PayResult) => void;
    onhold: (label: string) => void;
    onlock: () => void;
    onsync: () => void;
    onswitch: () => void;
  } = $props();

  const bridge = desktop();
  let which = $state<Modal>('none');
  let pendingSupervisor: (() => void) | null = $state(null);

  // price check
  let pcCode = $state('');
  let pcResult = $state<PriceCheckResult | null>(null);
  let pcBusy = $state(false);

  // quantity / discount
  let qtyValue = $state('');
  let discValue = $state('');

  // supervisor
  let supError = $state('');
  let supBusy = $state(false);

  // unknown barcode
  let unknownCode = $state('');

  export function open(m: Modal, code = '') {
    which = m;
    if (m === 'pricecheck') {
      pcCode = code;
      pcResult = null;
    }
    if (m === 'quantity') qtyValue = String(cart.rows[cart.selected]?.qty ?? 1);
    if (m === 'discount') discValue = '';
    if (m === 'unknown') unknownCode = code;
  }

  export function close() {
    which = 'none';
  }

  export function withSupervisor(fn: () => void) {
    pendingSupervisor = fn;
    supError = '';
    which = 'supervisor';
  }

  async function supervisorPin(pin: string) {
    if (!bridge) return;
    supBusy = true;
    try {
      const res = await bridge.Authorize('pos.supervisor.authorize', pin);
      if (res.grant) cart.grant = res.grant; // consumed by the next Pay
      const fn = pendingSupervisor;
      pendingSupervisor = null;
      close();
      fn?.();
    } catch (e) {
      supError = e instanceof Error ? e.message : 'Not authorised';
    } finally {
      supBusy = false;
    }
  }

  async function applyQty() {
    const n = Number(qtyValue);
    if (n > 0) await cart.setQty(cart.selected, n);
    close();
  }

  async function applyDiscount() {
    await cart.setDiscount(cart.selected, discValue || '0');
    close();
  }

  async function runPriceCheck() {
    if (!bridge || !pcCode.trim()) return;
    pcBusy = true;
    pcResult = await bridge.PriceCheck(pcCode.trim()).catch(() => null);
    pcBusy = false;
  }
</script>

{#if which === 'pay'}
  <PayModal {currency} {method} onclose={close} {onpaid} />
{:else if which === 'held'}
  <HeldModal onclose={close} {onhold} />
{:else if which === 'lookup'}
  <div class="overlay" role="presentation" onclick={(e) => e.target === e.currentTarget && close()}>
    <LookupModal {currency} onclose={close} />
  </div>
{:else if which !== 'none'}
  <div class="overlay" role="presentation" onclick={(e) => e.target === e.currentTarget && close()}>
    {#if which === 'pricecheck'}
      <div class="modal">
        <h3>Price check</h3>
        <div class="pcRow">
          <input
            bind:value={pcCode}
            placeholder="Barcode or SKU"
            onkeydown={(e) => e.key === 'Enter' && runPriceCheck()}
          />
          <button class="go" onclick={runPriceCheck} disabled={pcBusy}>{pcBusy ? '…' : 'Check'}</button>
        </div>
        {#if pcResult}
          {#if pcResult.unknown}
            <p class="muted">No product matches <code>{pcResult.code}</code>.</p>
          {:else}
            <div class="pcRes">
              <b>{pcResult.name}</b>
              <span class="sku">{pcResult.sku}</span>
              <span class="pcPrice">
                {currency}
                {formatMoney(pcResult.price ?? '0')}{pcResult.sell_by === 'weight' ? ' /kg' : ''}
              </span>
            </div>
          {/if}
        {/if}
        <button class="ghost" onclick={close}>Close</button>
      </div>
    {:else if which === 'quantity'}
      <ValuePrompt
        title="Quantity — {cart.rows[cart.selected]?.name ?? ''}"
        bind:value={qtyValue}
        submitLabel="Set quantity"
        onsubmit={applyQty}
        oncancel={close}
      />
    {:else if which === 'discount'}
      <ValuePrompt
        title="Line discount — {cart.rows[cart.selected]?.name ?? ''}"
        bind:value={discValue}
        placeholder="Amount off (e.g. 20)"
        onsubmit={applyDiscount}
        oncancel={close}
      />
    {:else if which === 'supervisor'}
      <div class="modal">
        <PinPad
          title="Supervisor authorisation"
          subtitle="This action needs a supervisor PIN"
          error={supError}
          busy={supBusy}
          onsubmit={supervisorPin}
          oncancel={close}
        />
      </div>
    {:else if which === 'more'}
      <MoreModal
        onclose={close}
        {onlock}
        {onsync}
        {onswitch}
        onheld={() => open('held')}
        onlookup={() => open('lookup')}
      />
    {:else if which === 'unknown'}
      <div class="modal">
        <h3>Unknown barcode</h3>
        <p class="muted">
          <code>{unknownCode}</code> isn't in the catalog. Check the label or sync the catalog.
        </p>
        <div class="row2">
          <button class="ghost" onclick={close}>Dismiss</button>
          <button
            class="go"
            onclick={() => {
              onsync();
              close();
            }}
          >
            Sync catalog
          </button>
        </div>
      </div>
    {/if}
  </div>
{/if}

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
    width: min(420px, 92vw);
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
  .pcRow {
    display: flex;
    gap: var(--space-2);
    margin-bottom: var(--space-3);
  }
  .pcRes {
    display: flex;
    flex-direction: column;
    gap: 2px;
    margin: var(--space-3) 0;
  }
  .pcRes .sku {
    color: var(--color-text-muted);
    font-size: var(--text-sm);
  }
  .pcPrice {
    font-size: var(--text-2xl);
    font-weight: 700;
    color: var(--color-primary);
  }
  .row2 {
    display: flex;
    gap: var(--space-2);
    margin-top: var(--space-3);
  }
  .row2 .go,
  .row2 .ghost {
    flex: 1;
  }
  .go {
    padding: var(--space-3) var(--space-4);
    font-weight: 700;
    color: #fff;
    background: var(--color-primary);
    border: none;
    border-radius: var(--radius-md);
    cursor: pointer;
  }
  .go:disabled {
    opacity: 0.5;
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
  .row2 .ghost {
    margin-top: 0;
  }
</style>
