<script lang="ts">
  import { onMount } from 'svelte';
  import { ApiError } from '@mauzo/api-client';
  import type { LoginHistoryEntry, MFAEnabled, MFASetup, SessionInfo } from '@mauzo/types';
  import { Button, DataTable, MfaPanel, StatusDot, toast } from '@mauzo/ui';
  import { formatDate, formatTime } from '@mauzo/utils';
  import { api } from '#lib/api.ts';
  import { auth, AUTH_BASE } from '#lib/auth.svelte.ts';

  let sessions = $state<SessionInfo[]>([]);
  let history = $state<LoginHistoryEntry[]>([]);
  let loading = $state(true);

  const sessionCols = [
    { key: 'ip', label: 'IP address', width: '140px' },
    { key: 'agent', label: 'Device' },
    { key: 'created_at', label: 'Signed in', width: '150px' },
    { key: 'last_used_at', label: 'Last used', width: '150px' },
    { key: 'actions', label: '', width: '110px', align: 'right' as const },
  ];
  const historyCols = [
    { key: 'created_at', label: 'Time', width: '150px' },
    { key: 'identity', label: 'Identity' },
    { key: 'result', label: 'Result', width: '140px' },
    { key: 'ip_address', label: 'IP', width: '120px' },
  ];

  const errText = (e: unknown) => (e instanceof ApiError ? e.message : 'Request failed');

  const mfa = {
    setup: () => api.post<MFASetup>(`${AUTH_BASE}/mfa/setup`),
    enable: async (code: string) =>
      (await api.post<MFAEnabled>(`${AUTH_BASE}/mfa/enable`, { code }))?.recovery_codes,
    disable: async (password: string) => {
      await api.post(`${AUTH_BASE}/mfa/disable`, { password });
    },
  };

  async function load() {
    loading = true;
    try {
      const [ss, hh] = await Promise.all([
        api.get<SessionInfo[]>(`${AUTH_BASE}/sessions`),
        api.get<LoginHistoryEntry[]>(`${AUTH_BASE}/history`),
      ]);
      sessions = ss ?? [];
      history = hh ?? [];
    } catch (e) {
      toast.error(errText(e));
    } finally {
      loading = false;
    }
  }

  async function revoke(s: SessionInfo) {
    try {
      await api.del(`${AUTH_BASE}/sessions/${s.id}`);
      sessions = sessions.filter((x) => x.id !== s.id);
      toast.success('Session revoked');
    } catch (e) {
      toast.error(errText(e));
    }
  }

  function agent(ua: string | undefined): string {
    if (!ua) return '—';
    return ua.length > 60 ? ua.slice(0, 57) + '…' : ua;
  }

  onMount(load);
</script>

<section class="head">
  <h1>Account</h1>
  <p class="muted">{auth.user?.name} · {auth.user?.role.replace('_', ' ')} · {auth.user?.identity}</p>
</section>

<div class="stack">
  <section class="card">
    <MfaPanel
      enabled={auth.user?.mfa_enabled ?? false}
      {...mfa}
      errorText={errText}
      onchanged={() => auth.refresh()}
    />
  </section>

  <section class="card">
    <h2>Active sessions</h2>
    <DataTable columns={sessionCols} rows={sessions} rowKey={(r) => r.id} {loading} empty="No sessions.">
      {#snippet cell(row: SessionInfo, col)}
        {#if col.key === 'ip'}
          {row.ip_address ?? '—'}{#if row.current}
            <span class="current">(this device)</span>{/if}
        {:else if col.key === 'agent'}
          {agent(row.user_agent)}
        {:else if col.key === 'created_at'}
          {formatDate(row.created_at)} {formatTime(row.created_at)}
        {:else if col.key === 'last_used_at'}
          {row.last_used_at ? `${formatDate(row.last_used_at)} ${formatTime(row.last_used_at)}` : '—'}
        {:else if col.key === 'actions' && !row.current}
          <Button size="sm" variant="danger" onclick={() => revoke(row)}>Revoke</Button>
        {/if}
      {/snippet}
    </DataTable>
  </section>

  <section class="card">
    <h2>Sign-in history</h2>
    <DataTable
      columns={historyCols}
      rows={history}
      rowKey={(r) => r.created_at + r.identity}
      {loading}
      empty="No history."
    >
      {#snippet cell(row: LoginHistoryEntry, col)}
        {#if col.key === 'created_at'}
          {formatDate(row.created_at)} {formatTime(row.created_at)}
        {:else if col.key === 'result'}
          <StatusDot
            status={row.success ? 'ok' : 'error'}
            label={row.success ? 'success' : (row.failure_reason ?? 'failed')}
          />
        {:else}
          {(row as unknown as Record<string, string>)[col.key] ?? '—'}
        {/if}
      {/snippet}
    </DataTable>
  </section>
</div>

<style>
  .head h1 {
    margin: 0 0 var(--space-1);
  }
  .muted {
    color: var(--color-text-muted);
    margin: 0;
  }
  .stack {
    display: flex;
    flex-direction: column;
    gap: var(--space-5);
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
    gap: var(--space-4);
  }
  .card h2 {
    margin: 0;
    font-size: var(--text-lg);
  }
  .current {
    color: var(--color-text-muted);
    font-size: var(--text-xs);
  }
</style>
