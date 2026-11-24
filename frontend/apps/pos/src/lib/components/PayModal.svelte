<script lang="ts">
  // Payment drawer from the right, over the product grid, so the cart stays in
  // view: amount due (big), method cards, the method's panel, Complete.
  // Cash is one tap on the note given; M-Pesa opens on the payments that just
  // reached this till. Keys: F1–F4 method, Enter complete, Esc back.
  import { Icon } from '@mauzo/ui';
  import { onMount, untrack } from 'svelte';
  import { cart } from '../cart.svelte';
  import { desktop, type PayResult } from '../desktop';
  import { formatMoney } from '../money';
  import { mpesafeed } from '../mpesafeed.svelte';
  import CashPanel from '../pay/CashPanel.svelte';
  import CustomerLine from '../pay/CustomerLine.svelte';
  import DrawerHeader from '../pay/DrawerHeader.svelte';
  import MpesaPanel from '../pay/MpesaPanel.svelte';
  import OtherPanel from '../pay/OtherPanel.svelte';
  import MethodTabs from '../pay/MethodTabs.svelte';
  import Summary from '../pay/Summary.svelte';
  import { paymethods } from '../pay/methods.svelte';
  import { type Method, Tender, fromCents, toCents } from '../pay/tender.svelte';
  import { pulse } from '../pulse.svelte';

  let {
    currency = 'KSh',
    method = 'cash',
    onclose,
    onpaid,
  }: {
    currency?: string;
    method?: string;
    onclose: () => void;
    onpaid: (r: PayResult) => void;
  } = $props();

  const bridge = desktop();
  const tender = new Tender(untrack(() => toCents(cart.totals.total)));
  const methods = $derived(paymethods.list);
  let active = $state<Method>(untrack(() => paymethods.pick(method)));
  let busy = $state(false);
  let error = $state('');
  let cash = $state<CashPanel | null>(null);

  const online = $derived(pulse.status?.online ?? true);
  const needsCustomer = $derived(tender.lines.some((l) => l.method === 'credit') && !cart.customerID);

  onMount(async () => {
    await paymethods.load();
    active = paymethods.pick(active);
    // A payment picked from the M-Pesa feed list is applied straight away.
    const ref = cart.mpesaRef;
    const t = ref ? mpesafeed.items.find((x) => x.trans_id === ref) : undefined;
    if (t?.trans_id) {
      tender.take('mpesa', toCents(t.amount), t.trans_id, t.payer_name || t.phone);
      active = 'mpesa';
    }
    cart.mpesaRef = '';
  });

  async function complete() {
    if (!bridge || busy || !tender.settled || needsCustomer) return;
    busy = true;
    error = '';
    try {
      const result = await bridge.Pay(
        JSON.stringify(tender.payload()),
        cart.payload(),
        '0',
        cart.customerID,
        cart.grant,
        '',
      );
      for (const l of tender.lines)
        if (l.method === 'mpesa' && l.reference) mpesafeed.dropByTrans(l.reference);
      // The drawer closes; the change and receipt show in the empty cart.
      onpaid(result);
    } catch (e) {
      error = e instanceof Error ? e.message : 'Payment failed';
    } finally {
      busy = false;
    }
  }

  function close() {
    if (!busy && (tender.lines.length === 0 || confirm('Discard the payments taken so far?'))) onclose();
  }

  // Capture phase so the checkout's own shortcuts (Esc = cancel sale) never
  // fire underneath this screen.
  function onkey(e: KeyboardEvent) {
    const typing = e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement;
    const fkeys: Record<string, number> = { F1: 0, F2: 1, F3: 2, F4: 3 };
    if (e.key in fkeys) {
      const m = methods[fkeys[e.key]!];
      if (m) active = m;
    } else if (e.key === 'Escape') {
      close();
    } else if (e.key === 'Enter') {
      if (tender.settled) void complete();
      else if (active === 'cash') cash?.submit();
      else return;
    } else if (!typing && active === 'cash' && /^([0-9.]|Backspace)$/.test(e.key)) {
      cash?.type(e.key);
    } else {
      return;
    }
    e.preventDefault();
    e.stopPropagation();
  }

  onMount(() => {
    window.addEventListener('keydown', onkey, true);
    return () => window.removeEventListener('keydown', onkey, true);
  });
</script>

<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
<div class="backdrop" onclick={(e) => e.target === e.currentTarget && close()}>
  <div class="drawer" role="dialog" aria-modal="true" aria-labelledby="pay-title">
    <DrawerHeader title="Complete payment" onclose={close} />

    <div class="body">
      <Summary {tender} {currency} />
      {#if tender.remaining === 0}
        <div class="paid">
          <strong>Fully paid</strong>
          <p>Press <kbd>Enter</kbd> to complete. Remove a payment above to change it.</p>
        </div>
      {:else}
        <MethodTabs {methods} bind:active />
        <div class="panel">
          {#if active === 'cash'}
            <CashPanel bind:this={cash} {tender} {currency} />
          {:else if active === 'mpesa'}
            <MpesaPanel {tender} {currency} {online} />
          {:else}
            {#key active}<OtherPanel {tender} {currency} method={active} />{/key}
          {/if}
        </div>
      {/if}
      <CustomerLine required={needsCustomer} />
    </div>

    <footer>
      {#if needsCustomer}<p class="err">Pick the customer to charge their account.</p>{/if}
      {#if error}<p class="err">{error}</p>{/if}
      <button class="complete" disabled={!tender.settled || busy || needsCustomer} onclick={complete}>
        {#if busy}
          Completing…
        {:else if tender.settled}
          <Icon name="check" size={22} />
          Complete sale{tender.change > 0 ? ` · change ${formatMoney(fromCents(tender.change))}` : ''}
          <kbd>Enter</kbd>
        {:else}
          {currency} {formatMoney(fromCents(tender.remaining))} still to pay
        {/if}
      </button>
      <button class="back" onclick={close}>
        <Icon name="arrow-left" size={18} /> Back to products <kbd>Esc</kbd>
      </button>
    </footer>
  </div>
</div>

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    z-index: 50;
    display: flex;
    justify-content: flex-end;
    background: rgb(15 23 42 / 0.45);
  }
  .drawer {
    width: min(660px, 100vw);
    height: 100%;
    display: flex;
    flex-direction: column;
    background: var(--color-bg);
    box-shadow: -18px 0 48px rgb(0 0 0 / 0.3);
    animation: slide 0.16s ease-out;
  }
  .body {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    gap: 14px;
    padding: 16px 18px;
    overflow-y: auto;
  }
  .panel {
    padding: 14px;
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-lg);
  }
  footer {
    display: grid;
    gap: 8px;
    padding: 12px 18px 16px;
    background: var(--color-surface);
    border-top: 1px solid var(--color-border);
  }
  .err {
    margin: 0;
    color: var(--color-danger);
    font-weight: 600;
  }
  .complete,
  .back {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    font: inherit;
    font-weight: 800;
    cursor: pointer;
  }
  .complete {
    min-height: 62px;
    font-size: 1.3rem;
    color: #fff;
    background: var(--color-success);
    border: 0;
    border-radius: var(--radius-lg);
  }
  .complete:disabled {
    color: var(--color-text-muted);
    background: var(--color-surface-muted);
    cursor: default;
  }
  .back {
    height: 50px;
    color: var(--color-primary);
    background: color-mix(in srgb, var(--color-primary) 6%, var(--color-surface));
    border: 1px solid color-mix(in srgb, var(--color-primary) 30%, transparent);
    border-radius: var(--radius-md);
  }
  kbd {
    font-size: 0.72rem;
    font-weight: 600;
    opacity: 0.65;
  }
  .paid {
    padding: 24px;
    text-align: center;
    border-radius: var(--radius-lg);
    background: color-mix(in srgb, var(--color-success) 10%, var(--color-surface));
  }
  .paid strong {
    font-size: 1.7rem;
    color: var(--color-success);
  }
  @keyframes slide {
    from {
      transform: translateX(40px);
      opacity: 0;
    }
  }
</style>
