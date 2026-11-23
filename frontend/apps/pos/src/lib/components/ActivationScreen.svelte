<script lang="ts">
  import { desktop, type ActivateResult } from '../desktop';

  let { onactivated }: { onactivated: (r: ActivateResult) => void } = $props();

  const bridge = desktop();
  let apiUrl = $state('http://localhost:8080');
  let code = $state('');
  let deviceName = $state('');
  let error = $state('');
  let busy = $state(false);

  async function activate() {
    if (!bridge) return;
    error = '';
    if (!apiUrl || code.length < 6 || !deviceName) {
      error = 'Fill in the server address, device name and activation code.';
      return;
    }
    busy = true;
    try {
      const r = await bridge.Activate({ api_url: apiUrl, code, device_name: deviceName });
      onactivated(r);
    } catch (e) {
      error = e instanceof Error ? e.message : 'Activation failed';
    } finally {
      busy = false;
    }
  }
</script>

<div class="screen">
  <div class="card">
    <h1>Activate this terminal</h1>
    <p class="muted">
      One-time setup. Get the activation code from the back office: Settings → POS terminals → Issue code.
    </p>
    <label>
      Server address
      <input bind:value={apiUrl} placeholder="http://server:8080" />
    </label>
    <label>
      Device name
      <input bind:value={deviceName} placeholder="Till 1 — front counter" />
    </label>
    <label>
      Activation code
      <input bind:value={code} placeholder="e.g. ab12cd34ef56" autocomplete="off" />
    </label>
    {#if error}<p class="err">{error}</p>{/if}
    <button class="go" onclick={activate} disabled={busy}>
      {busy ? 'Activating…' : 'Activate terminal'}
    </button>
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
    width: min(460px, 92vw);
    padding: var(--space-6);
    background: var(--color-surface);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-md, 0 4px 16px rgb(0 0 0 / 0.1));
  }
  h1 {
    margin: 0 0 var(--space-2);
  }
  .muted {
    color: var(--color-text-muted);
    margin-bottom: var(--space-4);
  }
  label {
    display: block;
    font-weight: 600;
    margin: var(--space-3) 0 0;
    font-size: var(--text-sm);
  }
  input {
    width: 100%;
    margin-top: var(--space-1);
    padding: var(--space-3);
    font: inherit;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    box-sizing: border-box;
  }
  .err {
    color: var(--color-danger);
  }
  .go {
    width: 100%;
    margin-top: var(--space-4);
    padding: var(--space-3);
    font-size: var(--text-lg);
    font-weight: 700;
    color: #fff;
    background: var(--color-primary);
    border: none;
    border-radius: var(--radius-md);
    cursor: pointer;
  }
  .go:disabled {
    opacity: 0.6;
  }
</style>
