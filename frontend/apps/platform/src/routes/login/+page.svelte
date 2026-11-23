<script lang="ts">
  import { ApiError } from '@mauzo/api-client';
  import { Button, TextField } from '@mauzo/ui';
  import { goto } from '$app/navigation';
  import { resolve } from '$app/paths';
  import { auth } from '#lib/auth.svelte.ts';

  let email = $state('');
  let password = $state('');
  let ticket = $state<string | null>(null);
  let code = $state('');
  let loading = $state(false);
  let error = $state<string | null>(null);

  async function submit() {
    loading = true;
    error = null;
    try {
      const res = await auth.login(email.trim(), password);
      if (res.mfa_required && res.ticket) {
        ticket = res.ticket;
        return;
      }
      goto(resolve('/'));
    } catch (e) {
      error = e instanceof ApiError ? e.message : 'Sign-in failed.';
    } finally {
      loading = false;
    }
  }

  async function verify() {
    loading = true;
    error = null;
    try {
      await auth.verifyMFA(ticket!, code.trim());
      goto(resolve('/'));
    } catch (e) {
      error = e instanceof ApiError ? e.message : 'Verification failed.';
    } finally {
      loading = false;
    }
  }
</script>

<div class="wrap">
  <form
    class="card"
    onsubmit={(e) => {
      e.preventDefault();
      if (ticket) verify();
      else submit();
    }}
  >
    <div class="brand">
      <strong>Mauzo</strong>
      <span>Platform Administration</span>
    </div>

    {#if ticket === null}
      <h1>Sign in</h1>
      <TextField label="Email" type="email" bind:value={email} required autocomplete="username" />
      <TextField
        label="Password"
        type="password"
        bind:value={password}
        required
        autocomplete="current-password"
      />
      {#if error}<p class="error">{error}</p>{/if}
      <Button type="submit" {loading} block>Sign in</Button>
      <a class="link" href={resolve('/forgot-password')}>Forgot password?</a>
    {:else}
      <h1>Two-factor code</h1>
      <p class="muted">Enter the 6-digit code from your authenticator app, or a recovery code.</p>
      <TextField label="Code" bind:value={code} required inputmode="numeric" autocomplete="one-time-code" />
      {#if error}<p class="error">{error}</p>{/if}
      <Button type="submit" {loading} block>Verify</Button>
      <button
        class="link"
        type="button"
        onclick={() => {
          ticket = null;
          code = '';
          error = null;
        }}
      >
        Back
      </button>
    {/if}
  </form>
</div>

<style>
  .wrap {
    display: grid;
    place-items: center;
    min-height: 100vh;
    padding: var(--space-4);
  }
  .card {
    width: 100%;
    max-width: 380px;
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-md);
    padding: var(--space-6);
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
  }
  .brand {
    display: flex;
    flex-direction: column;
    margin-bottom: var(--space-2);
  }
  .brand strong {
    font-size: var(--text-2xl);
    color: var(--color-primary);
  }
  .brand span {
    font-size: var(--text-sm);
    color: var(--color-text-muted);
  }
  h1 {
    margin: 0;
    font-size: var(--text-xl);
  }
  .muted {
    margin: 0;
    color: var(--color-text-muted);
    font-size: var(--text-sm);
  }
  .error {
    margin: 0;
    color: var(--color-danger);
    font-size: var(--text-sm);
  }
  .link {
    align-self: center;
    color: var(--color-primary);
    font-size: var(--text-sm);
    background: none;
    border: none;
    cursor: pointer;
    text-decoration: none;
  }
  .link:hover {
    text-decoration: underline;
  }
</style>
