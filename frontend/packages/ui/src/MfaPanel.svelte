<script lang="ts">
  import Button from './Button.svelte';
  import StatusDot from './StatusDot.svelte';
  import TextField from './TextField.svelte';

  interface Props {
    /** Whether TOTP is currently enabled on the account. */
    enabled: boolean;
    /** Starts setup; resolves to the secret + provisioning URI. */
    setup: () => Promise<{ secret: string; otpauth_url: string } | undefined>;
    /** Confirms setup with a TOTP code; resolves to recovery codes. */
    enable: (code: string) => Promise<string[] | undefined>;
    /** Disables MFA after password confirmation. */
    disable: (password: string) => Promise<void>;
    /** Called after enable/disable so the parent can refresh the profile. */
    onchanged?: () => void;
    /** Maps a thrown error to a display message. */
    errorText: (e: unknown) => string;
  }

  let { enabled, setup, enable, disable, onchanged, errorText }: Props = $props();

  // idle → setup (secret + code) → codes (recovery codes) | disable (password)
  let step = $state<'idle' | 'setup' | 'codes' | 'disable'>('idle');
  let secret = $state<{ secret: string; otpauth_url: string } | null>(null);
  let codes = $state<string[]>([]);
  let code = $state('');
  let pw = $state('');
  let busy = $state(false);
  let error = $state<string | null>(null);

  async function start() {
    busy = true;
    error = null;
    try {
      const res = await setup();
      if (res) {
        secret = res;
        code = '';
        step = 'setup';
      }
    } catch (e) {
      error = errorText(e);
    } finally {
      busy = false;
    }
  }

  async function confirm() {
    busy = true;
    error = null;
    try {
      codes = (await enable(code.trim())) ?? [];
      step = 'codes';
      onchanged?.();
    } catch (e) {
      error = errorText(e);
    } finally {
      busy = false;
    }
  }

  async function off() {
    busy = true;
    error = null;
    try {
      await disable(pw);
      pw = '';
      step = 'idle';
      onchanged?.();
    } catch (e) {
      error = errorText(e);
    } finally {
      busy = false;
    }
  }

  function cancel() {
    step = 'idle';
    secret = null;
    code = '';
    pw = '';
    error = null;
  }
</script>

<div class="head">
  <h2>Two-factor authentication</h2>
  <StatusDot status={enabled ? 'ok' : 'warn'} label={enabled ? 'enabled' : 'not enabled'} />
</div>

{#if step === 'idle'}
  <p class="muted">
    {enabled
      ? 'Your account requires a TOTP code at sign-in.'
      : 'Add an authenticator app code requirement to your sign-in.'}
  </p>
  {#if enabled}
    <div><Button variant="secondary" onclick={() => (step = 'disable')}>Disable…</Button></div>
  {:else}
    <div><Button onclick={start} loading={busy}>Set up…</Button></div>
  {/if}
{:else if step === 'setup' && secret}
  <p class="muted">Add this secret to your authenticator app, then enter the 6-digit code.</p>
  <div class="secret"><code>{secret.secret}</code></div>
  <p class="muted uri">Provisioning URI: <code>{secret.otpauth_url}</code></p>
  <TextField label="Code" bind:value={code} inputmode="numeric" autocomplete="one-time-code" />
  <div class="row">
    <Button variant="secondary" onclick={cancel}>Cancel</Button>
    <Button onclick={confirm} loading={busy}>Enable</Button>
  </div>
{:else if step === 'codes'}
  <p class="muted">Two-factor is on. Save these recovery codes — each works once in place of a TOTP code.</p>
  <div class="codes">
    {#each codes as c (c)}<code>{c}</code>{/each}
  </div>
  <div><Button onclick={cancel}>Done</Button></div>
{:else}
  <p class="muted">Enter your password to disable two-factor authentication.</p>
  <TextField label="Password" type="password" bind:value={pw} autocomplete="current-password" />
  <div class="row">
    <Button variant="secondary" onclick={cancel}>Cancel</Button>
    <Button variant="danger" onclick={off} loading={busy}>Disable 2FA</Button>
  </div>
{/if}
{#if error}<p class="error">{error}</p>{/if}

<style>
  .head {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
  .head h2 {
    margin: 0;
    font-size: var(--text-lg);
  }
  .muted {
    margin: 0;
    color: var(--color-text-muted);
  }
  .uri {
    font-size: var(--text-xs);
    word-break: break-all;
  }
  .secret {
    padding: var(--space-3) var(--space-4);
    background: var(--color-surface-muted);
    border: 1px dashed var(--color-border);
    border-radius: var(--radius-sm);
    font-size: var(--text-lg);
    letter-spacing: 0.08em;
    width: fit-content;
  }
  .codes {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
    gap: var(--space-2);
  }
  .codes code {
    padding: var(--space-2);
    background: var(--color-surface-muted);
    border-radius: var(--radius-sm);
    text-align: center;
  }
  .row {
    display: flex;
    gap: var(--space-3);
  }
  .error {
    margin: 0;
    color: var(--color-danger);
    font-size: var(--text-sm);
  }
</style>
