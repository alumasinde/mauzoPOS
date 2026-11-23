<script lang="ts">
  import { onMount } from 'svelte';
  import { ApiError } from '@mauzo/api-client';
  import type { PermissionGroup, Role } from '@mauzo/types';
  import { Badge, Button, DataTable, toast, PageHeader } from '@mauzo/ui';
  import { api } from '#lib/api.ts';
  import { auth } from '#lib/auth.svelte.ts';
  import RoleEditor from '#lib/roles/RoleEditor.svelte';

  let rows = $state<Role[]>([]);
  let groups = $state<PermissionGroup[]>([]);
  let loading = $state(true);
  let editorOpen = $state(false);
  let target = $state<Role | null>(null);
  let busy = $state<string | null>(null);
  const canManage = $derived(auth.can('roles.role.manage'));

  const columns = [
    { key: 'name', label: 'Role' },
    { key: 'kind', label: '', width: '200px' },
    { key: 'permissions', label: 'Permissions', width: '120px', align: 'right' as const },
    { key: 'user_count', label: 'Employees', width: '110px', align: 'right' as const },
    { key: 'actions', label: '', width: '220px', align: 'right' as const },
  ];

  async function load() {
    loading = true;
    try {
      [rows, groups] = await Promise.all([
        api.get<Role[]>('/api/v1/t/roles'),
        api.get<PermissionGroup[]>('/api/v1/t/permissions'),
      ]);
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Failed to load roles');
    } finally {
      loading = false;
    }
  }

  function open(role: Role | null) {
    target = role;
    editorOpen = true;
  }

  async function act(role: Role, kind: 'reset' | 'delete') {
    const msg =
      kind === 'reset'
        ? `Reset “${role.name}” to its built-in permissions?`
        : `Delete the role “${role.name}”? This cannot be undone.`;
    if (!confirm(msg)) return;
    busy = role.id;
    try {
      if (kind === 'reset') await api.post(`/api/v1/t/roles/${role.id}/reset`);
      else await api.del(`/api/v1/t/roles/${role.id}`);
      toast.success(kind === 'reset' ? 'Role reset' : 'Role deleted');
      load();
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Action failed');
    } finally {
      busy = null;
    }
  }

  onMount(load);
</script>

<PageHeader
  title="Roles & permissions"
  sub="What each role can do. Employees get one role each."
  icon="shield"
>
  {#snippet actions()}
    {#if canManage}<Button onclick={() => open(null)}>New role</Button>{/if}
  {/snippet}
</PageHeader>

<DataTable {columns} {rows} rowKey={(r) => r.id} {loading} onrowclick={(r) => open(r)}>
  {#snippet cell(row: Role, col)}
    {#if col.key === 'name'}
      <strong>{row.name}</strong>
      {#if row.description}<div class="muted small">{row.description}</div>{/if}
    {:else if col.key === 'kind'}
      {#if row.is_locked}
        <Badge tone="info">Locked</Badge>
      {:else if row.is_system}
        <Badge>Built-in</Badge>
        {#if row.customized}<Badge tone="warning">Customized</Badge>{/if}
      {:else}
        <Badge tone="success">Custom</Badge>
      {/if}
    {:else if col.key === 'permissions'}
      {row.is_locked ? 'All' : row.permissions.length}
    {:else if col.key === 'user_count'}
      {row.user_count}
    {:else if col.key === 'actions' && canManage && !row.is_locked}
      <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
      <div class="actions" onclick={(e) => e.stopPropagation()}>
        {#if row.is_system && row.customized}
          <Button size="sm" variant="secondary" loading={busy === row.id} onclick={() => act(row, 'reset')}>
            Reset
          </Button>
        {/if}
        {#if !row.is_system}
          <Button
            size="sm"
            variant="danger"
            loading={busy === row.id}
            disabled={row.user_count > 0}
            title={row.user_count > 0 ? 'Reassign its employees first' : undefined}
            onclick={() => act(row, 'delete')}
          >
            Delete
          </Button>
        {/if}
      </div>
    {/if}
  {/snippet}
</DataTable>

<RoleEditor bind:open={editorOpen} role={target} {groups} onsaved={load} />

<style>
  .muted {
    color: var(--color-text-muted);
    margin: 0;
  }
  .small {
    font-size: var(--text-sm);
  }
  .actions {
    display: flex;
    gap: var(--space-2);
    justify-content: flex-end;
  }
</style>
