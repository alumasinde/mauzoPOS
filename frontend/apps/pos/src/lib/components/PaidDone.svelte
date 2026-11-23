<script lang="ts">
  // Post-tender confirmation — receipt number + change, or the offline
  // queued state with the idempotency key.
  import type { PayResult } from '../desktop';
  import { formatMoney } from '../money';

  let {
    currency = 'KSh',
    result,
    ondone,
  }: {
    currency?: string;
    result: PayResult;
    ondone: () => void;
  } = $props();
</script>

<h3>{result.offline ? 'Sale queued' : 'Sale complete'}</h3>
<div class="donebox">
  {#if result.offline}
    <p class="muted">Offline — the sale will upload on the next sync. Idempotency key:</p>
    <code class="key">{result.sale_id}</code>
  {:else}
    <div class="rcpt">Receipt <b>#{result.receipt_no}</b></div>
    <div class="tot">Total <b>{currency} {formatMoney(result.total)}</b></div>
    {#if Number(result.change_due) > 0}
      <div class="chg">Change due <b>{currency} {formatMoney(result.change_due)}</b></div>
    {/if}
  {/if}
</div>
<button class="go" onclick={ondone}>Done — new sale</button>

<style>
  h3 { margin: 0 0 var(--space-3); font-size: var(--text-lg); }
  .donebox { text-align: center; padding: var(--space-4) 0; }
  .muted { color: var(--color-text-muted); font-size: var(--text-sm); }
  .key { display: block; margin-top: var(--space-2); font-size: var(--text-xs); word-break: break-all; }
  .rcpt { font-size: var(--text-xl); margin-bottom: var(--space-2); }
  .tot { font-size: var(--text-lg); }
  .chg { margin-top: var(--space-2); color: var(--color-primary); font-size: var(--text-lg); }
  .go { width: 100%; padding: 12px; border: 0; background: var(--color-success); color: #fff; font-weight: 700; border-radius: var(--radius-md); cursor: pointer; font-size: var(--text-base); }
</style>
