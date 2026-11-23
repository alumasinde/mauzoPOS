<script lang="ts">
  import { onMount } from 'svelte';
  import { desktop, type TerminalSettings } from '../desktop';

  let { onback }: { onback: () => void } = $props();

  const bridge = desktop();
  let s = $state<TerminalSettings>({
    device_name: '',
    api_url: '',
    printer_kind: 'none',
    printer_name: '',
    printer_host: '',
    printer_port: 9100,
    auto_print: 'default',
    paper_width: 80,
    auto_lock_seconds: 120,
    business_type: '',
    sync_cursor: '',
  });
  let msg = $state('');
  let err = $state('');
  let busy = $state(false);
  let printers = $state<string[]>([]);
  let testing = $state(false);

  onMount(async () => {
    if (!bridge) return;
    s = await bridge.GetSettings();
    if (!s.printer_port) s.printer_port = 9100;
    printers = await bridge.ListPrinters().catch(() => []);
  });

  // Save first so the test uses what is on screen.
  async function testPrint() {
    if (!bridge) return;
    testing = true;
    err = '';
    try {
      await bridge.SaveSettings(s);
      await bridge.TestPrint();
      msg = 'Test receipt sent to the printer.';
      setTimeout(() => (msg = ''), 4000);
    } catch (e) {
      err = e instanceof Error ? e.message : 'Test print failed';
    } finally {
      testing = false;
    }
  }

  async function kickDrawer() {
    if (!bridge) return;
    err = '';
    try {
      await bridge.OpenDrawer();
    } catch (e) {
      err = e instanceof Error ? e.message : 'Could not open the drawer';
    }
  }

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
    <fieldset>
      <legend>Receipt printer</legend>
      <label>
        Connection
        <select bind:value={s.printer_kind}>
          <option value="none">No printer</option>
          <option value="windows">Installed in Windows (USB / Bluetooth)</option>
          <option value="network">Network (Ethernet / Wi-Fi)</option>
          {#if s.printer_kind === 'file'}<option value="file">File (support)</option>{/if}
        </select>
      </label>
      {#if s.printer_kind === 'windows'}
        <label>
          Printer
          <select bind:value={s.printer_name}>
            <option value="">Choose a printer…</option>
            {#each printers as p (p)}<option value={p}>{p}</option>{/each}
            {#if s.printer_name && !printers.includes(s.printer_name)}
              <option value={s.printer_name}>{s.printer_name} (not found)</option>
            {/if}
          </select>
        </label>
        <p class="hint">
          Install the printer's driver (or Windows "Generic / Text Only"). Receipts are sent raw, so driver
          paper settings do not matter.
        </p>
      {:else if s.printer_kind === 'network'}
        <div class="two">
          <label>IP address <input bind:value={s.printer_host} placeholder="192.168.1.50" /></label>
          <label>Port <input type="number" min="1" max="65535" bind:value={s.printer_port} /></label>
        </div>
      {/if}
      <label>
        Print after each sale
        <select bind:value={s.auto_print}>
          <option value="default">Use the business setting</option>
          <option value="always">Always print</option>
          <option value="never">Only when the cashier presses Print</option>
        </select>
      </label>
      <p class="hint">Receipt layout: {s.paper_width} mm (set in the back office).</p>
      {#if s.printer_kind !== 'none'}
        <div class="row tight">
          <button class="ghost" onclick={testPrint} disabled={testing}>
            {testing ? 'Printing…' : 'Print test receipt'}
          </button>
          <button class="ghost" onclick={kickDrawer}>Open cash drawer</button>
        </div>
      {/if}
    </fieldset>
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
    max-height: 94vh;
    overflow-y: auto;
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
  fieldset {
    margin: var(--space-4) 0 0;
    padding: var(--space-3) var(--space-4) var(--space-4);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
  }
  legend {
    font-weight: 700;
    padding: 0 var(--space-1);
  }
  .two {
    display: grid;
    grid-template-columns: 1fr 110px;
    gap: var(--space-2);
  }
  .hint {
    margin: var(--space-2) 0 0;
    font-size: var(--text-xs);
    color: var(--color-text-muted);
  }
  .row.tight {
    justify-content: flex-start;
    margin-top: var(--space-3);
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
