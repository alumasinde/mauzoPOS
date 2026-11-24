<script lang="ts">
  // M-Pesa: the till's live "received" list first (most customers pay to the
  // till number), then the phone prompt and the SMS-code fallback.
  import type { MpesaTxn } from '../desktop';
  import MpesaCode from './MpesaCode.svelte';
  import MpesaPrompt from './MpesaPrompt.svelte';
  import MpesaReceived from './MpesaReceived.svelte';
  import { type Tender, toCents } from './tender.svelte';

  let { tender, currency, online }: { tender: Tender; currency: string; online: boolean } = $props();

  let mode = $state<'received' | 'prompt' | 'code'>('received');

  function use(t: MpesaTxn) {
    if (!t.trans_id || tender.hasReference(t.trans_id)) return;
    tender.take('mpesa', toCents(t.amount), t.trans_id, t.payer_name || t.phone);
    mode = 'received';
  }
</script>

<div class="mpesa">
  <nav aria-label="M-Pesa options">
    <button class:on={mode === 'received'} onclick={() => (mode = 'received')}>Paid to till</button>
    <button class:on={mode === 'prompt'} onclick={() => (mode = 'prompt')} disabled={!online}>
      Send prompt
    </button>
    <button class:on={mode === 'code'} onclick={() => (mode = 'code')}>Enter code</button>
  </nav>

  {#if mode === 'received'}
    <MpesaReceived {tender} {currency} onuse={use} />
  {:else if mode === 'prompt'}
    <MpesaPrompt {tender} {currency} onpaid={use} />
  {:else}
    <MpesaCode
      {tender}
      {currency}
      {online}
      ontake={(code, cents) => {
        tender.take('mpesa', cents, code, 'Code entered by cashier');
        mode = 'received';
      }}
    />
  {/if}
</div>

<style>
  .mpesa {
    display: grid;
    gap: 16px;
  }
  nav {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 6px;
    padding: 4px;
    border-radius: var(--radius-md);
    background: var(--color-surface-muted);
  }
  nav button {
    height: 44px;
    font: inherit;
    font-weight: 700;
    color: var(--color-text-muted);
    background: none;
    border: 0;
    border-radius: var(--radius-sm);
    cursor: pointer;
  }
  nav button.on {
    color: var(--color-text);
    background: var(--color-surface);
    box-shadow: var(--shadow-sm);
  }
  nav button:disabled {
    opacity: 0.4;
    cursor: default;
  }
</style>
