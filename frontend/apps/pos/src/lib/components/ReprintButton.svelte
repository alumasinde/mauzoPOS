<script lang="ts">
  // Audited receipt reprint. A cashier without pos.receipt.reprint is asked
  // for a supervisor PIN; the server numbers the copy and logs it.
  import { desktop } from '../desktop';
  import PinPad from './PinPad.svelte';

  const PERM = 'pos.receipt.reprint';

  let { receipt, label = 'Reprint receipt' }: { receipt: string | number; label?: string } = $props();

  const bridge = desktop();
  let busy = $state(false);
  let asking = $state(false);
  let pinError = $state('');
  let msg = $state('');
  let err = $state('');

  async function start() {
    if (!bridge || busy) return;
    msg = '';
    err = '';
    if (await bridge.Can(PERM)) return run('');
    pinError = '';
    asking = true;
  }

  async function supervisor(pin: string) {
    if (!bridge) return;
    busy = true;
    try {
      const res = await bridge.Authorize(PERM, pin);
      if (!res.grant) {
        pinError = 'Reprints need the server — try again when online.';
        return;
      }
      asking = false;
      await run(res.grant);
    } catch (e) {
      pinError = e instanceof Error ? e.message : 'Wrong PIN';
    } finally {
      busy = false;
    }
  }

  async function run(grant: string) {
    if (!bridge) return;
    busy = true;
    try {
      const res = await bridge.Reprint(String(receipt), grant);
      if (res.printed) msg = `Copy ${res.copy_no} printed`;
      else err = `Copy ${res.copy_no} recorded but not printed: ${res.error ?? 'printer error'}`;
    } catch (e) {
      err = e instanceof Error ? e.message : 'Reprint failed';
    } finally {
      busy = false;
    }
  }
</script>

<div class="reprint">
  <button class="ghost" onclick={start} disabled={busy}>{busy ? 'Printing…' : label}</button>
  {#if msg}<span class="ok">{msg}</span>{/if}
  {#if err}<span class="err">{err}</span>{/if}
</div>

{#if asking}
  <div class="overlay">
    <div class="pinbox">
      <PinPad
        title="Supervisor authorisation"
        subtitle="Reprinting a receipt needs a supervisor PIN"
        error={pinError}
        {busy}
        onsubmit={supervisor}
        oncancel={() => (asking = false)}
      />
    </div>
  </div>
{/if}

<style>
  .reprint {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
    align-items: stretch;
  }
  .ghost {
    padding: 10px;
    font: inherit;
    font-weight: 600;
    background: none;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    cursor: pointer;
    color: var(--color-text);
  }
  .ok {
    color: var(--color-success);
    font-size: var(--text-sm);
    text-align: center;
  }
  .err {
    color: var(--color-danger);
    font-size: var(--text-sm);
    text-align: center;
  }
  .overlay {
    position: fixed;
    inset: 0;
    display: grid;
    place-items: center;
    background: rgb(0 0 0 / 0.5);
    z-index: 60;
  }
  .pinbox {
    width: min(420px, 92vw);
    background: var(--color-surface);
    border-radius: var(--radius-lg);
    padding: var(--space-5);
  }
</style>
