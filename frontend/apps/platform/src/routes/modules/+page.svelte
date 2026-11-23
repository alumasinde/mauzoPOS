<script lang="ts">
  import { onMount } from 'svelte';
  import { ApiError } from '@mauzo/api-client';
  import type { ModuleInfo } from '@mauzo/types';
  import { Badge, DataTable, Switch, toast } from '@mauzo/ui';
  import { api } from '#lib/api.ts';
  import { auth } from '#lib/auth.svelte.ts';

  let rows = $state<ModuleInfo[]>([]);
  let loading = $state(true);
  let busy = $state<string | null>(null);
  const canManage = $derived(auth.can('platform.module.manage'));

  const columns = [
    { key: 'name', label: 'Module' },
    { key: 'group', label: 'Group', width: '120px' },
    { key: 'depends', label: 'Depends on', width: '220px' },
    { key: 'permissions', label: 'Permissions', width: '120px', align: 'right' as const },
    { key: 'active', label: 'Available', width: '110px', align: 'right' as const },
  ];

  async function load() {
    loading = true;
    try {
      rows = await api.get<ModuleInfo[]>('/api/v1/platform/modules');
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Failed to load modules');
    } finally {
      loading = false;
    }
  }

  async function setActive(m: ModuleInfo, active: boolean) {
    if (!active && !confirm(`Withdraw ${m.name} from every tenant? Their settings are kept.`)) return;
    busy = m.code;
    try {
      rows = await api.patch<ModuleInfo[]>(`/api/v1/platform/modules/${m.code}`, { active });
      toast.success(`${m.name} ${active ? 'available again' : 'withdrawn platform-wide'}`);
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Update failed');
    } finally {
      busy = null;
    }
  }

  onMount(load);
</script>

<section class="head">
  <h1>Module registry</h1>
  <p class="muted">
    Modules are defined in code. Switching one off here withdraws it from every tenant immediately (their
    configuration is kept). Core modules cannot be withdrawn.
  </p>
</section>

<DataTable {columns} {rows} rowKey={(r) => r.code} {loading}>
  {#snippet cell(row: ModuleInfo, col)}
    {#if col.key === 'name'}
      <strong>{row.name}</strong>
      {#if row.core}<Badge tone="info">Core</Badge>{/if}
      <div class="muted small">{row.description}</div>
    {:else if col.key === 'group'}
      {row.group}
    {:else if col.key === 'depends'}
      {row.depends.join(', ') || '—'}
    {:else if col.key === 'permissions'}
      {row.permissions.length}
    {:else if col.key === 'active'}
      <Switch
        checked={!!row.platform_active}
        label={`Availability of ${row.name}`}
        disabled={row.core || !canManage}
        busy={busy === row.code}
        onchange={(on) => setActive(row, on)}
      />
    {/if}
  {/snippet}
</DataTable>

<style>
  .head h1 {
    margin: 0 0 var(--space-1);
  }
  .muted {
    color: var(--color-text-muted);
    margin: 0 0 var(--space-5);
  }
  .small {
    font-size: var(--text-sm);
    margin: 0;
  }
</style>
