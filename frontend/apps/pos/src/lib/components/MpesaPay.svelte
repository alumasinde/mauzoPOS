<script lang="ts">
  import { Icon } from '@mauzo/ui';
  // STK-push row for PayModal: enter the customer's phone, send the PIN
  // prompt, poll until Daraja reports the result, then hand the receipt code
  // back to the payment row as its reference.
  import { desktop, type MpesaTxn } from '../desktop';
  import { formatMoney } from '../money';

  let {
    amount,
    accountRef = 'SALE',
    onconfirmed,
    onfailed,
  }: {
    amount: string;
    accountRef?: string;
    onconfirmed: (transID: string) => void;
    onfailed: () => void;
  } = $props();

  const bridge = desktop();
  let phone = $state('');
  let phase = $state<'idle' | 'pushing' | 'waiting' | 'done' | 'failed'>('idle');
  let txn = $state<MpesaTxn | null>(null);
  let message = $state('');
  let timer: ReturnType<typeof setTimeout> | null = null;

  const validPhone = $derived(/^(\+?254|0)(7|1)\d{8}$/.test(phone.replace(/[\s-]/g, '')));

  async function push() {
    if (!bridge || !validPhone || phase === 'pushing') return;
    phase = 'pushing';
    message = '';
    try {
      txn = await bridge.MpesaPush(phone, amount, accountRef);
      phase = 'waiting';
      poll();
    } catch (e) {
      phase = 'idle';
      message = e instanceof Error ? e.message : 'Could not send the prompt';
    }
  }

  function poll() {
    timer = setTimeout(async () => {
      if (!bridge || !txn) return;
      try {
        txn = await bridge.MpesaStatus(txn.public_id);
      } catch {
        poll(); // transient — keep waiting
        return;
      }
      if (txn.status === 'success') {
        phase = 'done';
        if (txn.trans_id) onconfirmed(txn.trans_id);
      } else if (txn.status === 'pending' || txn.status === 'initiated') {
        poll();
      } else {
        phase = 'failed';
        message =
          txn.status === 'cancelled'
            ? 'The customer dismissed the prompt.'
            : txn.status === 'timeout'
              ? 'No result — the prompt expired.'
              : txn.result_desc || 'The payment failed.';
        onfailed();
      }
    }, 2500);
  }

  function reset() {
    if (timer) clearTimeout(timer);
    txn = null;
    phase = 'idle';
    message = '';
  }

  $effect(() => () => {
    if (timer) clearTimeout(timer);
  });
</script>

<div class="mpesa">
  {#if phase === 'done' && txn}
    <div class="ok">
      <Icon name="check-circle" size={18} />
      <span>
        Paid <b>{txn.trans_id}</b> — KSh {formatMoney(txn.amount)}
      </span>
    </div>
  {:else}
    <div class="row">
      <input
        bind:value={phone}
        inputmode="tel"
        placeholder="Customer phone — 07XXXXXXXX"
        disabled={phase === 'waiting' || phase === 'pushing'}
        onkeydown={(e) => e.key === 'Enter' && push()}
      />
      <button
        class="send"
        onclick={push}
        disabled={!validPhone || phase === 'pushing' || phase === 'waiting'}
      >
        {phase === 'pushing' ? 'Sending…' : 'Send prompt'}
      </button>
    </div>
    {#if phase === 'waiting'}
      <div class="wait">
        <span class="spin"><Icon name="refresh" size={16} /></span>
        Waiting for the customer to enter their PIN — KSh {formatMoney(amount)}
      </div>
    {/if}
    {#if phase === 'failed'}
      <div class="fail">
        <Icon name="alert" size={16} />
        <span>{message}</span>
        <button class="retry" onclick={reset}>Try again</button>
      </div>
    {:else if message}
      <p class="err">{message}</p>
    {/if}
    <p class="hint">…or type the M-Pesa code in the reference field above.</p>
  {/if}
</div>

<style>
  .mpesa {
    grid-column: 1 / -1;
    background: color-mix(in srgb, var(--color-primary) 6%, transparent);
    border: 1px solid color-mix(in srgb, var(--color-primary) 25%, transparent);
    border-radius: var(--radius-md);
    padding: var(--space-3);
    margin-top: var(--space-1);
  }
  .row {
    display: flex;
    gap: var(--space-2);
  }
  input {
    flex: 1;
    padding: 10px;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    font: inherit;
    background: var(--color-surface);
  }
  .send {
    padding: 10px 14px;
    border: 0;
    border-radius: var(--radius-md);
    background: #00a651;
    color: #fff;
    font-weight: 700;
    cursor: pointer;
    white-space: nowrap;
  }
  .send:disabled {
    opacity: 0.5;
    cursor: default;
  }
  .wait {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    margin-top: var(--space-2);
    font-size: var(--text-sm);
    color: var(--color-primary);
  }
  .spin {
    display: inline-flex;
    animation: rot 1s linear infinite;
  }
  @keyframes rot {
    to {
      transform: rotate(360deg);
    }
  }
  .ok {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    color: var(--color-success);
    font-weight: 600;
  }
  .fail {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    margin-top: var(--space-2);
    font-size: var(--text-sm);
    color: var(--color-danger);
  }
  .retry {
    margin-left: auto;
    border: 1px solid var(--color-border);
    background: var(--color-surface);
    border-radius: var(--radius-md);
    padding: 4px 10px;
    font-size: var(--text-sm);
    cursor: pointer;
  }
  .err {
    color: var(--color-danger);
    font-size: var(--text-sm);
    margin: var(--space-2) 0 0;
  }
  .hint {
    margin: var(--space-2) 0 0;
    font-size: var(--text-xs);
    color: var(--color-text-muted);
  }
</style>
