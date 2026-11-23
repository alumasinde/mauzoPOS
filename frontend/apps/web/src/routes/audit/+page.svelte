<script lang="ts">
  import { onMount } from 'svelte';
  import { ApiError } from '@mauzo/api-client';
  import type { AuditEntry } from '@mauzo/types';
  import { DataTable, TextField, toast, PageHeader } from '@mauzo/ui';
  import { formatDate, formatTime } from '@mauzo/utils';
  import { api } from '#lib/api.ts';

  let rows = $state<AuditEntry[]>([]);
  let loading = $state(true);
  let q = $state('');
  let pageNum = $state(1);
  let total = $state(0);

  const columns = [
    { key: 'created_at', label: 'Time', width: '150px' },
    { key: 'actor', label: 'Actor', width: '160px' },
    { key: 'action', label: 'Action' },
    { key: 'entity', label: 'Entity' },
    { key: 'ip_address', label: 'IP', width: '120px' },
  ];

  async function load() {
    loading = true;
    try {
      const res = await api.list<AuditEntry>('/api/v1/t/audit-logs', { page: pageNum, q });
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

<PageHeader title="Audit log" sub="{total} entries" icon="list">
  {#snippet actions()}
    <TextField
      placeholder="Search action / entity…"
      bind:value={q}
      oninput={() => {
        pageNum = 1;
        load();
      }}
    />
  {/snippet}
</PageHeader>

<DataTable {columns} {rows} rowKey={(r) => r.id} {loading} empty="No audit entries.">
  {#snippet cell(row: AuditEntry, col)}
    {#if col.key === 'created_at'}
      {formatDate(row.created_at)} {formatTime(row.created_at)}
    {:else if col.key === 'actor'}
      {row.actor_type}
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
