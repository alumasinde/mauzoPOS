<script lang="ts">
  // "Lipa kwa simu": send a PIN prompt to the customer's phone for the balance
  // (rounded up to whole shillings — M-Pesa has no cents). The result arrives
  // on the live feed; a slow status poll is only the fallback.
  import { Icon } from '@mauzo/ui';
  import { cart } from '../cart.svelte';
  import { desktop, type MpesaTxn } from '../desktop';
  import { formatMoney } from '../money';
  import { mpesafeed } from '../mpesafeed.svelte';
  import PhoneForm from './PhoneForm.svelte';
  import { type Tender, fromCents } from './tender.svelte';

  let { tender, currency, onpaid }: { tender: Tender; currency: string; onpaid: (t: MpesaTxn) => void } =
    $props();

  const bridge = desktop();
  let phone = $state('');
  let elapsed = $state(0);
  let tick: ReturnType<typeof setInterval> | undefined;
  let phase = $state<'idle' | 'sending' | 'waiting' | 'failed'>('idle');
  let txn = $state<MpesaTxn | null>(null);
  let message = $state('');
  let timer: ReturnType<typeof setTimeout> | undefined;

  // Whole shillings, rounded up: the excess cents come back as change. The
  // cashier may ask for less (a split: part M-Pesa, part cash), never more.
  let custom = $state('');
  const due = $derived(Math.ceil(tender.remaining / 100) * 100);
  const amount = $derived(custom.trim() ? Math.min(due, Math.floor(Number(custom) || 0) * 100) : due);
  const clean = $derived(phone.replace(/\D/g, ''));
  const phoneOk = $derived(/^(254|0)(7|1)\d{8}$/.test(clean));
  const valid = $derived(phoneOk && amount >= 100);
  const local = (p: string) => p.replace(/\D/g, '').replace(/^254/, '0');

  function settle(t: MpesaTxn) {
    if (!txn || t.public_id !== txn.public_id) return;
    txn = t;
    if (t.status === 'success' && t.trans_id) {
      stop();
      phase = 'idle';
      onpaid(t);
    } else if (t.status !== 'pending' && t.status !== 'initiated') {
      stop();
      phase = 'failed';
      message =
        t.status === 'cancelled'
          ? 'The customer cancelled the prompt.'
          : t.status === 'timeout'
            ? 'No answer — the prompt expired.'
            : t.result_desc || 'The payment did not go through.';
    }
  }

  const unwatch = mpesafeed.onStk(settle);
  $effect(() => () => {
    unwatch();
    stop();
  });

  function stop() {
    if (timer) clearTimeout(timer);
    if (tick) clearInterval(tick);
    timer = tick = undefined;
  }

  // Stop waiting here; if the customer still pays, it lands in "Paid to till".
  function giveUp() {
    stop();
    txn = null;
    phase = 'idle';
    message = 'Stopped waiting. If the customer pays now, pick it under "Paid to till".';
  }

  function poll() {
    timer = setTimeout(async () => {
      if (!bridge || !txn || phase !== 'waiting') return;
      try {
        settle(await bridge.MpesaStatus(txn.public_id));
      } catch {
        /* transient — keep waiting */
      }
      if (phase === 'waiting') poll();
    }, 5000);
  }

  async function send() {
    if (!bridge || !valid || phase === 'sending' || phase === 'waiting') return;
    phase = 'sending';
    message = '';
    try {
      txn = await bridge.MpesaPush(clean, fromCents(amount), 'SALE');
      phase = 'waiting';
      elapsed = 0;
      tick = setInterval(() => elapsed++, 1000);
      poll();
    } catch (e) {
      phase = 'idle';
      message = e instanceof Error ? e.message : 'Could not send the prompt';
    }
  }
</script>

<section>
  {#if phase === 'waiting'}
    <div class="wait">
      <span class="spin"><Icon name="refresh" size={30} /></span>
      <div>
        <strong>{currency} {formatMoney(fromCents(amount))} requested from {local(clean)}</strong>
        <p>Ask the customer to check their phone and enter their M-Pesa PIN. This completes by itself.</p>
        <small>Waiting {elapsed}s</small>
      </div>
    </div>
    <button class="cancel" onclick={giveUp}>Stop waiting</button>
  {:else}
    <PhoneForm
      bind:phone
      bind:custom
      {due}
      {amount}
      {currency}
      {phoneOk}
      sending={phase === 'sending'}
      {message}
      customerPhone={local(cart.customerPhone)}
      customerName={cart.customerName}
      onsend={send}
    />
  {/if}
</section>

<style>
  section {
    display: grid;
    gap: 12px;
  }
  .wait {
    display: flex;
    gap: 16px;
    align-items: center;
    padding: 22px;
    border-radius: var(--radius-md);
    background: color-mix(in srgb, var(--color-success) 10%, var(--color-surface));
    border: 1px solid color-mix(in srgb, var(--color-success) 35%, transparent);
  }
  .wait strong {
    font-size: 1.2rem;
  }
  .wait p {
    margin: 4px 0 0;
    color: var(--color-text-muted);
  }
  .wait small {
    color: var(--color-text-muted);
  }
  .cancel {
    height: 48px;
    font: inherit;
    font-weight: 700;
    color: var(--color-text);
    background: none;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    cursor: pointer;
  }
  .spin {
    color: var(--color-success);
    animation: spin 1.1s linear infinite;
  }
  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }
</style>
