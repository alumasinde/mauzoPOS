<script lang="ts">
  // The STK form: customer's number (typed, or one tap on the picked
  // customer's), the amount (defaults to the balance), and Send.
  import { Icon } from '@mauzo/ui';
  import { onMount } from 'svelte';
  import { formatMoney } from '../money';
  import Keypad from './Keypad.svelte';
  import { fromCents } from './tender.svelte';

  let {
    phone = $bindable(),
    custom = $bindable(),
    due,
    amount,
    currency,
    phoneOk,
    sending,
    message,
    customerPhone,
    customerName,
    onsend,
  }: {
    phone: string;
    custom: string;
    due: number;
    amount: number;
    currency: string;
    phoneOk: boolean;
    sending: boolean;
    message: string;
    customerPhone: string;
    customerName: string;
    onsend: () => void;
  } = $props();

  let input = $state<HTMLInputElement | null>(null);
  let pad = $state(false);
  const digits = $derived(phone.replace(/\D/g, ''));
  const bad = $derived(digits.length >= 10 && !phoneOk);

  onMount(() => {
    if (!phone && customerPhone) phone = format(customerPhone);
    input?.focus();
  });

  // 0712345678 → 0712 345 678; 254712345678 → 254 712 345 678.
  function format(v: string): string {
    const d = v.replace(/\D/g, '').slice(0, 12);
    const parts = d.startsWith('254') ? [3, 3, 3, 3] : [4, 3, 3];
    const out: string[] = [];
    let i = 0;
    for (const n of parts) {
      if (i >= d.length) break;
      out.push(d.slice(i, i + n));
      i += n;
    }
    return out.join(' ');
  }
</script>

<form
  onsubmit={(e) => {
    e.preventDefault();
    onsend();
  }}
>
  <label class="field" class:bad>
    <span class="lbl">Customer phone number</span>
    <span class="box">
      <Icon name="phone" size={22} />
      <input
        bind:this={input}
        value={phone}
        oninput={(e) => (phone = format(e.currentTarget.value))}
        inputmode="tel"
        autocomplete="off"
        placeholder="07XX XXX XXX"
        aria-invalid={bad}
      />
      {#if phone}
        <button type="button" class="clr" aria-label="Clear number" onclick={() => (phone = '')}>
          <Icon name="x" size={16} />
        </button>
      {/if}
    </span>
    {#if bad}<small class="hint err">Use a Safaricom number like 0712 345 678 or 0110 123 456.</small>{/if}
  </label>

  <div class="chips">
    {#if customerPhone && format(customerPhone) !== phone}
      <button type="button" class="chip" onclick={() => (phone = format(customerPhone))}>
        <Icon name="user" size={15} />
        {customerName || 'Customer'} · {format(customerPhone)}
      </button>
    {/if}
    <button type="button" class="chip ghost" onclick={() => (pad = !pad)}>
      <Icon name="grid" size={15} />
      {pad ? 'Hide keypad' : 'Keypad'}
    </button>
  </div>
  {#if pad}<Keypad bind:value={() => digits, (v) => (phone = format(v))} decimal={false} max={12} />{/if}

  <label class="field">
    <span class="lbl">Amount</span>
    <span class="box">
      <span class="cur">{currency}</span>
      <input inputmode="numeric" placeholder={formatMoney(fromCents(due))} bind:value={custom} />
    </span>
    {#if amount < due}<small class="hint">Part payment — the rest stays to pay by another method.</small>{/if}
  </label>

  <p class="info">
    <Icon name="info" size={18} />
    The customer gets an M-Pesa PIN prompt on their phone. The payment shows here by itself once they enter it.
  </p>

  {#if message}<p class="msg"><Icon name="alert" size={16} /> {message}</p>{/if}

  <button class="send" type="submit" disabled={!phoneOk || amount < 100 || sending}>
    <Icon name="phone" size={22} />
    {sending ? 'Sending…' : `Send M-Pesa request · ${currency} ${formatMoney(fromCents(amount))}`}
  </button>
</form>

<style>
  form {
    display: grid;
    gap: 12px;
  }
  .field {
    display: grid;
    gap: 6px;
  }
  .lbl {
    font-weight: 700;
  }
  .box {
    display: flex;
    align-items: center;
    gap: 10px;
    height: 58px;
    padding: 0 14px;
    color: var(--color-primary);
    background: var(--color-surface);
    border: 2px solid var(--color-border);
    border-radius: var(--radius-md);
  }
  .box:focus-within {
    border-color: var(--color-primary);
  }
  .bad .box {
    border-color: var(--color-danger);
  }
  input {
    flex: 1;
    min-width: 0;
    height: 100%;
    font: inherit;
    font-size: 1.6rem;
    font-weight: 700;
    letter-spacing: 0.04em;
    font-variant-numeric: tabular-nums;
    color: var(--color-text);
    background: none;
    border: 0;
    outline: none;
  }
  .cur {
    padding: 4px 8px;
    font-size: 0.85rem;
    font-weight: 800;
    color: var(--color-text-muted);
    background: var(--color-surface-muted);
    border-radius: 6px;
  }
  .clr {
    display: grid;
    place-items: center;
    width: 30px;
    height: 30px;
    color: var(--color-text-muted);
    background: var(--color-surface-muted);
    border: 0;
    border-radius: 50%;
    cursor: pointer;
  }
  .hint {
    color: var(--color-text-muted);
  }
  .hint.err {
    color: var(--color-danger);
  }
  .chips {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }
  .chip {
    display: flex;
    gap: 6px;
    align-items: center;
    padding: 8px 12px;
    font: inherit;
    font-weight: 700;
    color: var(--color-primary);
    background: color-mix(in srgb, var(--color-primary) 8%, var(--color-surface));
    border: 1px solid color-mix(in srgb, var(--color-primary) 30%, transparent);
    border-radius: 999px;
    cursor: pointer;
  }
  .chip.ghost {
    color: var(--color-text-muted);
    background: none;
    border-color: var(--color-border);
  }
  .info {
    display: flex;
    gap: 10px;
    align-items: flex-start;
    margin: 0;
    padding: 12px 14px;
    color: var(--color-text);
    background: color-mix(in srgb, var(--color-primary) 7%, var(--color-surface));
    border-radius: var(--radius-md);
  }
  .info :global(svg) {
    flex: none;
    color: var(--color-primary);
  }
  .msg {
    display: flex;
    gap: 6px;
    align-items: center;
    margin: 0;
    color: var(--color-danger);
    font-weight: 600;
  }
  .send {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 10px;
    height: 62px;
    font: inherit;
    font-size: 1.2rem;
    font-weight: 800;
    color: #fff;
    background: #1f8f4e;
    border: 0;
    border-radius: var(--radius-md);
    cursor: pointer;
  }
  .send:disabled {
    opacity: 0.45;
    cursor: default;
  }
</style>
