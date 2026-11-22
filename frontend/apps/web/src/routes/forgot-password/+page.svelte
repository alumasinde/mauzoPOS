<script lang="ts">
  import { ApiError } from '@mauzo/api-client';
  import { Button, TextField } from '@mauzo/ui';
  import { onMount } from 'svelte';
  import { resolve } from '$app/paths';
  import { api } from '#lib/api.ts';
  import { auth } from '#lib/auth.ts';

  let slug = $state('');
  let identity = $state('');
  let loading = $state(false);
  let sent = $state(false);
  let error = $state<string | null>(null);

  onMount(() => {
    slug = auth.tenantSlug;
  });

  async function submit() {
    loading = true;
    error = null;
    try {
      await api.post('/api/v1/t/auth/password/forgot', {
        tenant: slug.trim(),
        identity: identity.trim(),
      });
      sent = true;
    } catch (e) {
      error = e instanceof ApiError ? e.message : 'Request failed.';
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
      submit();
    }}
  >
    <h1>Reset password</h1>
    {#if sent}
      <p class="muted">If the details match an account, a reset link is on its way.</p>
      <a class="link" href={resolve('/login')}>Back to sign in</a>
    {:else}
      <p class="muted">Enter your company code and email or username.</p>
      <TextField label="Company code" bind:value={slug} required autocapitalize="off" />
      <TextField label="Email or username" bind:value={identity} required />
      {#if error}<p class="error">{error}</p>{/if}
      <Button type="submit" {loading} block>Send reset link</Button>
      <a class="link" href={resolve('/login')}>Back to sign in</a>
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
  }
</style>
