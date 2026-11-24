<script lang="ts">
  // Checkout modals (price check, quantity, discount, supervisor PIN, more,
  // unknown barcode); open() also routes to the pay and held components.
  import { desktop, type PayResult } from '../desktop';
  import { cart } from '../cart.svelte';
  import PinPad from './PinPad.svelte';
  import PayModal from './PayModal.svelte';
  import HeldModal from './HeldModal.svelte';
  import ValuePrompt from './ValuePrompt.svelte';
  import MoreModal from './MoreModal.svelte';
  import LookupModal from './LookupModal.svelte';
  import MpesaFeedModal from './MpesaFeedModal.svelte';
  import ShiftFlow from './ShiftFlow.svelte';
  import UnknownBarcode from './UnknownBarcode.svelte';
  import PriceCheck from './PriceCheck.svelte';

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
    | 'lookup'
    | 'mpesa'
    | 'shiftopen'
    | 'shiftcash'
    | 'shiftclose'
    | 'shiftreport';

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
  // The pay modal's method defaults to the action bar's; a feed pick forces mpesa.
  let payMethod = $state('');

  // price check
  let pcCode = $state('');

  // quantity / discount
  let qtyValue = $state('');
  let discValue = $state('');

  // supervisor
  let supError = $state('');
  let supBusy = $state(false);

  // unknown barcode
  let unknownCode = $state('');

  export function open(m: Modal, code = '') {
    if (m === 'pay') payMethod = '';
    which = m;
    if (m === 'pricecheck') pcCode = code;
    if (m === 'quantity') qtyValue = String(cart.rows[cart.selected]?.qty ?? 1);
    if (m === 'discount') discValue = '';
    if (m === 'unknown') unknownCode = code;
  }

  /** A modal is showing — checkout shortcuts (Esc = cancel sale) must not fire. */
  export function isOpen(): boolean {
    return which !== 'none';
  }

  export function close() {
    which = 'none';
  }

  // A completed sale closes the drawer; the result shows in the empty cart.
  function paid(r: PayResult) {
    close();
    onpaid(r);
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
</script>

{#if which === 'pay'}
  <PayModal {currency} method={payMethod || method} onclose={close} onpaid={paid} />
{:else if which === 'mpesa'}
  <div class="overlay" role="presentation" onclick={(e) => e.target === e.currentTarget && close()}>
    <MpesaFeedModal
      onclose={close}
      onuse={() => {
        payMethod = 'mpesa';
        which = 'pay';
      }}
    />
  </div>
{:else if which === 'held'}
  <HeldModal onclose={close} {onhold} />
{:else if which === 'lookup'}
  <div class="overlay" role="presentation" onclick={(e) => e.target === e.currentTarget && close()}>
    <LookupModal {currency} onclose={close} />
  </div>
{:else if which !== 'none'}
  <div class="overlay" role="presentation" onclick={(e) => e.target === e.currentTarget && close()}>
    {#if which === 'pricecheck'}
      <PriceCheck code={pcCode} {currency} onclose={close} />
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
        {onswitch}
        onheld={() => open('held')}
        onlookup={() => open('lookup')}
        onmpesa={() => open('mpesa')}
        onshift={(m) => open(m)}
      />
    {:else if which === 'shiftopen' || which === 'shiftcash' || which === 'shiftclose' || which === 'shiftreport'}
      <ShiftFlow mode={which} {currency} onclose={close} />
    {:else if which === 'unknown'}
      <UnknownBarcode code={unknownCode} {onsync} onclose={close} />
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
</style>
