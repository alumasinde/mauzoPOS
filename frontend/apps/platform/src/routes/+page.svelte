<script lang="ts">
  import { onMount } from 'svelte';
  import { ApiError } from '@mauzo/api-client';
  import type { HealthStatus } from '@mauzo/types';
  import { StatusDot } from '@mauzo/ui';
  import { resolve } from '$app/paths';
  import { api } from '#lib/api.ts';
  import { auth } from '#lib/auth.svelte.ts';

  let health = $state<HealthStatus | null>(null);
  let error = $state<string | null>(null);

  onMount(async () => {
    try {
      health = await api.request<HealthStatus>('GET', '/readyz');
    } catch (e) {
      error =
        e instanceof ApiError && e.status === 503
          ? 'The API is up but a dependency is unavailable.'
          : e instanceof Error
            ? e.message
            : String(e);
    }
  });
</script>

<section class="head">
  <h1>Dashboard</h1>
  <p class="muted">Signed in as {auth.user?.name} ({auth.user?.role.replace('_', ' ')}).</p>
</section>

<div class="grid">
  <section class="card">
    <h2>System status</h2>
    {#if health}
      <StatusDot status={health.status === 'ok' ? 'ok' : 'warn'} label={health.status} />
      <span class="muted">v{health.version}</span>
      {#each Object.entries(health.checks ?? {}) as [name, state] (name)}
        <div class="row">
          <span>{name}</span>
          <StatusDot status={state === 'ok' ? 'ok' : 'error'} label={state} />
        </div>
      {/each}
    {:else}
      <StatusDot status="unknown" label="checking…" />
    {/if}
    {#if error}<p class="error">{error}</p>{/if}
  </section>

  <section class="card">
    <h2>Quick links</h2>
    <nav class="links">
      <a href={resolve('/tenants')}>Manage tenants</a>
      <a href={resolve('/users')}>Platform users</a>
      <a href={resolve('/audit')}>Audit log</a>
    </nav>
  </section>
</div>

<style>
  .head h1 {
    margin: 0 0 var(--space-1);
  }
  .muted {
    color: var(--color-text-muted);
  }
  .grid {
    display: grid;
    gap: var(--space-5);
    grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
    margin-top: var(--space-5);
  }
  .card {
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-lg);
    padding: var(--space-5);
    box-shadow: var(--shadow-sm);
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }
  .card h2 {
    margin: 0;
    font-size: var(--text-lg);
  }
  .row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    text-transform: capitalize;
  }
  .links {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }
  .links a {
    color: var(--color-primary);
  }
  .error {
    color: var(--color-danger);
    margin: 0;
  }
</style>
