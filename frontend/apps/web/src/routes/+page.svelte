<script lang="ts">
  import { onMount } from 'svelte';
  import { ApiError } from '@mauzo/api-client';
  import type { HealthStatus } from '@mauzo/types';
  import { Button, StatusDot } from '@mauzo/ui';
  import { formatDate, formatTime } from '@mauzo/utils';
  import { api } from '#lib/api.ts';

  let health = $state<HealthStatus | null>(null);
  let error = $state<string | null>(null);
  let loading = $state(false);
  let checkedAt = $state<Date | null>(null);

  async function check() {
    loading = true;
    error = null;
    try {
      health = await api.request<HealthStatus>('GET', '/readyz');
    } catch (e) {
      health = null;
      error =
        e instanceof ApiError && e.status === 503
          ? 'The API is up but a dependency is unavailable.'
          : e instanceof Error
            ? e.message
            : String(e);
    } finally {
      loading = false;
      checkedAt = new Date();
    }
  }

  onMount(check);
</script>

<section class="card">
  <h1>System status</h1>
  <p class="muted">Phase 0 foundation — this page verifies the app can reach the Go API.</p>

  <dl>
    <dt>API</dt>
    <dd>
      {#if health}
        <StatusDot status={health.status === 'ok' ? 'ok' : 'warn'} label={health.status} />
        <span class="muted">v{health.version}</span>
      {:else if error}
        <StatusDot status="error" label="unreachable" />
      {:else}
        <StatusDot status="unknown" label="checking…" />
      {/if}
    </dd>
    {#each Object.entries(health?.checks ?? {}) as [name, state] (name)}
      <dt>{name}</dt>
      <dd><StatusDot status={state === 'ok' ? 'ok' : 'error'} label={state} /></dd>
    {/each}
  </dl>

  {#if error}<p class="error">{error}</p>{/if}

  <footer>
    <Button variant="secondary" {loading} onclick={check}>Check again</Button>
    {#if checkedAt}
      <span class="muted">Last checked {formatDate(checkedAt)} {formatTime(checkedAt)}</span>
    {/if}
  </footer>
</section>

<style>
  .card {
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-lg);
    padding: var(--space-5);
    box-shadow: var(--shadow-sm);
    max-width: 560px;
  }
  h1 {
    margin: 0 0 var(--space-2);
    font-size: var(--text-xl);
  }
  .muted {
    color: var(--color-text-muted);
    font-size: var(--text-sm);
  }
  dl {
    display: grid;
    grid-template-columns: 120px 1fr;
    gap: var(--space-3);
    margin: var(--space-5) 0;
  }
  dt {
    font-weight: 600;
    text-transform: capitalize;
  }
  dd {
    margin: 0;
    display: flex;
    gap: var(--space-3);
    align-items: center;
  }
  .error {
    color: var(--color-danger);
  }
  footer {
    display: flex;
    align-items: center;
    gap: var(--space-4);
  }
</style>
