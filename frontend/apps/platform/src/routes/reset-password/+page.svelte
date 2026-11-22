<script lang="ts">
  import { ApiError } from '@mauzo/api-client';
  import { Button, TextField } from '@mauzo/ui';
  import { page } from '$app/state';
  import { resolve } from '$app/paths';
  import { api } from '#lib/api.ts';

  let password = $state('');
  let confirm = $state('');
  let loading = $state(false);
  let done = $state(false);
  let error = $state<string | null>(null);

  const token = $derived(page.url.searchParams.get('token') ?? '');

  async function submit() {
    if (password !== confirm) {
      error = 'Passwords do not match.';
      return;
    }
    loading = true;
    error = null;
    try {
      await api.post('/api/v1/platform/auth/password/reset', { token, password });
      done = true;
    } catch (e) {
      error = e instanceof ApiError ? e.message : 'Reset failed.';
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
    <h1>Choose a new password</h1>
    {#if !token}
      <p class="error">The reset link is missing its token.</p>
    {:else if done}
      <p class="muted">Your password was changed. All sessions were signed out.</p>
      <a class="link" href={resolve('/login')}>Sign in</a>
    {:else}
      <TextField
        label="New password"
        type="password"
        bind:value={password}
        required
        hint="At least 8 characters"
        autocomplete="new-password"
      />
      <TextField
        label="Confirm password"
        type="password"
        bind:value={confirm}
        required
        autocomplete="new-password"
      />
      {#if error}<p class="error">{error}</p>{/if}
      <Button type="submit" {loading} block>Set password</Button>
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
