<script lang="ts">
  import { onMount } from 'svelte';
  import { desktop, type TerminalSettings } from '../desktop';

  let { onback }: { onback: () => void } = $props();

  const bridge = desktop();
  let s = $state<TerminalSettings>({
    device_name: '',
    api_url: '',
    printer_name: '',
    auto_lock_seconds: 120,
    business_type: '',
    sync_cursor: '',
  });
  let msg = $state('');
  let err = $state('');
  let busy = $state(false);

  onMount(async () => {
    if (!bridge) return;
    s = await bridge.GetSettings();
  });

  async function save() {
    if (!bridge) return;
    busy = true;
    err = '';
    try {
      await bridge.SaveSettings(s);
      msg = 'Saved.';
      setTimeout(() => (msg = ''), 2500);
    } catch (e) {
      err = e instanceof Error ? e.message : 'Save failed';
    } finally {
      busy = false;
    }
  }
</script>

<div class="screen">
  <div class="card">
    <h1>Terminal settings</h1>
    <label>Device name <input bind:value={s.device_name} /></label>
    <label>Server address <input bind:value={s.api_url} /></label>
    <label>
      Receipt printer <input bind:value={s.printer_name} placeholder="Printer name (Phase 8 uses it)" />
    </label>
    <label>
      Auto-lock after
      <select bind:value={s.auto_lock_seconds}>
        <option value={60}>1 minute</option>
        <option value={120}>2 minutes</option>
        <option value={300}>5 minutes</option>
        <option value={600}>10 minutes</option>
      </select>
    </label>
    <dl>
      <dt>Business type</dt>
      <dd>{s.business_type || '—'}</dd>
      <dt>Last sync</dt>
      <dd>{s.sync_cursor || 'never'}</dd>
    </dl>
    {#if msg}<p class="ok">{msg}</p>{/if}
    {#if err}<p class="err">{err}</p>{/if}
    <div class="row">
      <button class="ghost" onclick={onback}>Back to checkout</button>
      <button class="go" onclick={save} disabled={busy}>{busy ? 'Saving…' : 'Save settings'}</button>
    </div>
  </div>
</div>

<style>
  .screen {
    display: grid;
    place-items: center;
    height: 100vh;
    background: var(--color-surface-muted);
  }
  .card {
    width: min(480px, 94vw);
    padding: var(--space-6);
    background: var(--color-surface);
    border-radius: var(--radius-lg);
  }
  h1 {
    margin-top: 0;
  }
  label {
    display: block;
    font-weight: 600;
    margin: var(--space-3) 0 0;
    font-size: var(--text-sm);
  }
  input,
  select {
    width: 100%;
    margin-top: var(--space-1);
    padding: var(--space-2);
    font: inherit;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    box-sizing: border-box;
  }
  dl {
    display: grid;
    grid-template-columns: 140px 1fr;
    margin: var(--space-4) 0;
  }
  dt {
    color: var(--color-text-muted);
  }
  dd {
    margin: 0;
  }
  .ok {
    color: var(--color-success);
  }
  .err {
    color: var(--color-danger);
  }
  .row {
    display: flex;
    gap: var(--space-2);
    justify-content: space-between;
    margin-top: var(--space-4);
  }
  .go {
    padding: var(--space-2) var(--space-5);
    font-weight: 700;
    color: #fff;
    background: var(--color-primary);
    border: none;
    border-radius: var(--radius-md);
    cursor: pointer;
  }
  .ghost {
    padding: var(--space-2) var(--space-4);
    font: inherit;
    background: none;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    cursor: pointer;
  }
</style>
