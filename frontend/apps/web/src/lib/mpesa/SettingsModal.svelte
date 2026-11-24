<script lang="ts">
  // Daraja configuration: masked credential fields (blanks keep the stored
  // secret), the webhook URLs Safaricom posts to, URL registration and a
  // 1-KES test prompt to prove the setup end to end.
  import { ApiError } from '@mauzo/api-client';
  import { Button, Icon, Modal, Select, TextField, toast } from '@mauzo/ui';
  import { onMount } from 'svelte';
  import { mpesaApi, type MpesaConfig } from './api';

  let { open = $bindable(false), onsaved }: { open: boolean; onsaved: () => void } = $props();

  let cfg = $state<MpesaConfig | null>(null);
  let form = $state({
    env: 'sandbox',
    short_code: '',
    short_code_kind: 'till',
    consumer_key: '',
    consumer_secret: '',
    passkey: '',
    initiator_name: '',
    security_credential: '',
  });
  let testPhone = $state('');
  let busy = $state(false);
  let busyAction = $state('');

  onMount(async () => {
    try {
      cfg = await mpesaApi.config();
      form = {
        env: cfg.env || 'sandbox',
        short_code: cfg.short_code,
        short_code_kind: cfg.short_code_kind || 'till',
        consumer_key: cfg.consumer_key,
        consumer_secret: cfg.consumer_secret,
        passkey: cfg.passkey,
        initiator_name: cfg.initiator_name,
        security_credential: cfg.security_credential,
      };
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Could not load M-Pesa settings');
    }
  });

  // Masked values ("••••xxxx") round-trip unchanged — the server ignores them.
  async function save() {
    busy = true;
    busyAction = 'save';
    try {
      await mpesaApi.saveConfig(form);
      toast.success('M-Pesa settings saved.');
      onsaved();
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Save failed');
    } finally {
      busy = false;
      busyAction = '';
    }
  }

  async function register() {
    busy = true;
    busyAction = 'register';
    try {
      await mpesaApi.registerC2B();
      toast.success('Callback URLs registered with Daraja.');
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Registration failed');
    } finally {
      busy = false;
      busyAction = '';
    }
  }

  async function test() {
    if (!testPhone.trim()) return;
    busy = true;
    busyAction = 'test';
    try {
      await mpesaApi.testPush(testPhone.trim());
      toast.success('Prompt sent — check the phone for the PIN request.');
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Test failed');
    } finally {
      busy = false;
      busyAction = '';
    }
  }

  function copy(v: string) {
    void navigator.clipboard.writeText(v);
    toast.info('Copied.');
  }
</script>

<Modal bind:open title="M-Pesa (Daraja) settings" size="lg">
  {#if cfg}
    <div class="cols">
      <div class="field">
        <label for="env">Environment</label>
        <Select id="env" bind:value={form.env}>
          <option value="sandbox">Sandbox</option>
          <option value="production">Production</option>
        </Select>
      </div>
      <div class="field">
        <label for="kind">Shortcode kind</label>
        <Select id="kind" bind:value={form.short_code_kind}>
          <option value="till">Buy Goods till</option>
          <option value="paybill">Pay Bill</option>
        </Select>
      </div>
      <TextField id="sc" label="Shortcode (till / paybill)" bind:value={form.short_code} />
      <TextField id="iname" label="Initiator name (reversals)" bind:value={form.initiator_name} />
      <TextField
        id="ck"
        label="Consumer key"
        bind:value={form.consumer_key}
        placeholder={cfg.has_consumer_secret ? '•••• — saved' : 'Daraja consumer key'}
      />
      <TextField
        id="cs"
        label="Consumer secret"
        type="password"
        bind:value={form.consumer_secret}
        placeholder={cfg.has_consumer_secret ? '•••• — saved' : 'Daraja consumer secret'}
      />
      <TextField
        id="pk"
        label="STK passkey"
        type="password"
        bind:value={form.passkey}
        placeholder={cfg.has_passkey ? '•••• — saved' : 'Lipa Na M-Pesa passkey'}
      />
      <TextField
        id="scred"
        label="Security credential (reversals)"
        type="password"
        bind:value={form.security_credential}
        placeholder={cfg.has_security_credential ? '•••• — saved' : 'Encrypted initiator password'}
      />
    </div>
    <p class="hint">Masked fields keep the stored secret unless you type a new value.</p>

    <h4>Webhook URLs</h4>
    <p class="hint">Register these on the Daraja portal (or use the button — it does it for you).</p>
    <div class="urls">
      {#each [['STK callback', cfg.stk_callback_url], ['C2B confirmation', cfg.c2b_confirmation_url], ['C2B validation', cfg.c2b_validation_url], ['Reversal result', cfg.reversal_result_url]] as [label, url] (label)}
        {#if url}
          <div class="urow">
            <span class="ulabel">{label}</span>
            <code>{url}</code>
            <button class="cp" onclick={() => copy(url)} aria-label="Copy">
              <Icon name="copy" size={14} />
            </button>
          </div>
        {/if}
      {/each}
      {#if !cfg.stk_callback_url}
        <p class="hint">Save credentials first — the URLs are generated from them.</p>
      {/if}
    </div>

    <div class="actions">
      <Button onclick={save} loading={busy && busyAction === 'save'}>Save settings</Button>
      <Button
        variant="secondary"
        onclick={register}
        loading={busy && busyAction === 'register'}
        disabled={!cfg.short_code}
      >
        Register callback URLs
      </Button>
      {#if cfg.c2b_registered_at}
        <span class="reg">URLs registered {cfg.c2b_registered_at.slice(0, 10)}</span>
      {/if}
    </div>

    <h4>Test the setup</h4>
    <div class="testrow">
      <input bind:value={testPhone} inputmode="tel" placeholder="Phone — 07XXXXXXXX" />
      <Button
        variant="secondary"
        onclick={test}
        loading={busy && busyAction === 'test'}
        disabled={!testPhone.trim()}
      >
        Send KSh 1 prompt
      </Button>
    </div>
  {:else}
    <p class="hint">Loading…</p>
  {/if}
</Modal>

<style>
  .cols {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 0 var(--space-3);
  }
  .field {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
    margin-bottom: var(--space-3);
  }
  .field label {
    font-size: var(--text-sm);
    font-weight: 600;
  }
  .hint {
    font-size: var(--text-xs);
    color: var(--color-text-muted);
    margin: var(--space-1) 0 var(--space-3);
  }
  h4 {
    margin: var(--space-4) 0 var(--space-2);
    font-size: var(--text-sm);
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--color-text-muted);
  }
  .urls {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    margin-bottom: var(--space-3);
  }
  .urow {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    font-size: var(--text-xs);
  }
  .ulabel {
    min-width: 130px;
    font-weight: 600;
  }
  .urow code {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    background: color-mix(in srgb, var(--color-primary) 6%, transparent);
    padding: 4px 8px;
    border-radius: var(--radius-md);
  }
  .cp {
    border: 0;
    background: none;
    color: var(--color-primary);
    cursor: pointer;
    padding: 4px;
  }
  .actions {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    flex-wrap: wrap;
  }
  .reg {
    font-size: var(--text-xs);
    color: var(--color-success);
  }
  .testrow {
    display: flex;
    gap: var(--space-2);
  }
  .testrow input {
    flex: 1;
    padding: 8px 10px;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    font: inherit;
  }
</style>
