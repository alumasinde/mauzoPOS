<script lang="ts">
  import { onMount } from 'svelte';
  import { ApiError } from '@mauzo/api-client';
  import type { AuditEntry } from '@mauzo/types';
  import { DataTable, TextField, toast } from '@mauzo/ui';
  import { formatDate, formatTime } from '@mauzo/utils';
  import { api } from '#lib/api.ts';

  let rows = $state<AuditEntry[]>([]);
  let loading = $state(true);
  let q = $state('');
  let actorType = $state('');
  let pageNum = $state(1);
  let total = $state(0);

  const columns = [
    { key: 'created_at', label: 'Time', width: '150px' },
    { key: 'actor_type', label: 'Actor', width: '120px' },
    { key: 'action', label: 'Action' },
    { key: 'entity', label: 'Entity' },
    { key: 'ip_address', label: 'IP', width: '120px' },
  ];

  async function load() {
    loading = true;
    try {
      const res = await api.list<AuditEntry>('/api/v1/platform/audit-logs', {
        page: pageNum,
        q,
        actor_type: actorType,
      });
      rows = res.data;
      total = res.meta.total;
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Failed to load audit log');
    } finally {
      loading = false;
    }
  }

  onMount(load);
</script>

<section class="head">
  <div>
    <h1>Audit log</h1>
    <p class="muted">{total} entries</p>
  </div>
  <div class="tools">
    <label class="field">
      <select
        bind:value={actorType}
        onchange={() => {
          pageNum = 1;
          load();
        }}
      >
        <option value="">All actors</option>
        <option value="platform_user">Platform users</option>
        <option value="user">Tenant users</option>
        <option value="system">System</option>
      </select>
    </label>
    <TextField
      placeholder="Search action / entity…"
      bind:value={q}
      oninput={() => {
        pageNum = 1;
        load();
      }}
    />
  </div>
</section>

<DataTable {columns} {rows} rowKey={(r) => r.id} {loading} empty="No audit entries.">
  {#snippet cell(row: AuditEntry, col)}
    {#if col.key === 'created_at'}
      {formatDate(row.created_at)} {formatTime(row.created_at)}
    {:else if col.key === 'entity'}
      {row.entity_type ?? ''}{row.entity_id ? `#${row.entity_id.slice(0, 8)}…` : ''}
    {:else}
      {(row as unknown as Record<string, string>)[col.key] ?? ''}
    {/if}
  {/snippet}
</DataTable>

<div class="pager">
  <button
    disabled={pageNum <= 1}
    onclick={() => {
      pageNum--;
      load();
    }}
  >
    Previous
  </button>
  <span>Page {pageNum}</span>
  <button
    disabled={rows.length < 25}
    onclick={() => {
      pageNum++;
      load();
    }}
  >
    Next
  </button>
</div>

<style>
  .head {
    display: flex;
    justify-content: space-between;
    align-items: end;
    gap: var(--space-4);
    margin-bottom: var(--space-5);
  }
  .head h1 {
    margin: 0 0 var(--space-1);
  }
  .muted {
    color: var(--color-text-muted);
    margin: 0;
  }
  .tools {
    display: flex;
    gap: var(--space-3);
    align-items: center;
  }
  select {
    min-height: 40px;
    padding: var(--space-2) var(--space-3);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    background: var(--color-surface);
    color: var(--color-text);
    font: inherit;
  }
  .pager {
    display: flex;
    align-items: center;
    gap: var(--space-4);
    justify-content: center;
    margin-top: var(--space-4);
  }
  .pager button {
    border: 1px solid var(--color-border);
    background: var(--color-surface);
    border-radius: var(--radius-sm);
    padding: var(--space-2) var(--space-4);
    font: inherit;
    cursor: pointer;
  }
  .pager button:disabled {
    opacity: 0.5;
    cursor: default;
  }
</style>
