<script lang="ts">
  import { desktop, type ActivateResult } from '../desktop';

  let { onactivated }: { onactivated: (r: ActivateResult) => void } = $props();

  const bridge = desktop();
  let apiUrl = $state('http://localhost:8080');
  let code = $state('');
  let deviceName = $state('');
  let error = $state('');
  let busy = $state(false);

  // Codes are 10 characters shown as ABCDE-FGH23; format while typing so the
  // cashier can compare with the back-office screen at a glance.
  function formatCode(raw: string): string {
    const c = raw
      .toUpperCase()
      .replace(/[^A-Z0-9]/g, '')
      .slice(0, 10);
    return c.length > 5 ? `${c.slice(0, 5)}-${c.slice(5)}` : c;
  }
  const ready = $derived(code.replace('-', '').length === 10);

  async function activate() {
    if (!bridge) return;
    error = '';
    if (!apiUrl || !ready || !deviceName.trim()) {
      error = 'Fill in the server address, device name and the 10-character activation code.';
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
      One-time setup. In the back office open <b>Team &amp; access → POS terminals</b>, choose this till and
      press <b>Get activation code</b>. Codes are valid for 24 hours.
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
      <input
        class="code"
        value={code}
        oninput={(e) => (code = e.currentTarget.value = formatCode(e.currentTarget.value))}
        onkeydown={(e) => e.key === 'Enter' && activate()}
        placeholder="ABCDE-FGH23"
        autocomplete="off"
        spellcheck="false"
      />
    </label>
    {#if error}<p class="err">{error}</p>{/if}
    <button class="go" onclick={activate} disabled={busy || !ready}>
      {busy ? 'Activating…' : 'Activate terminal'}
    </button>
  </div>
</div>

<style>
  .code {
    font-size: 1.6rem;
    font-weight: 800;
    letter-spacing: 0.12em;
    text-align: center;
    font-family: var(--font-mono, monospace);
  }
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
