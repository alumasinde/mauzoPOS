<script lang="ts">
  import { onMount } from 'svelte';
  import { ApiError } from '@mauzo/api-client';
  import type { Branch, Employee, Ref } from '@mauzo/types';
  import { Badge, Button, DataTable, Pager, StatusDot, TextField, toast } from '@mauzo/ui';
  import { formatDate } from '@mauzo/utils';
  import { api } from '#lib/api.ts';
  import { auth } from '#lib/auth.svelte.ts';
  import EmployeeForm from '#lib/employees/EmployeeForm.svelte';
  import SecretModal from '#lib/employees/SecretModal.svelte';

  const PER_PAGE = 25;
  let rows = $state<Employee[]>([]);
  let roles = $state<Ref[]>([]); // roles this user may assign
  let branches = $state<Branch[]>([]);
  let loading = $state(true);
  let q = $state('');
  let status = $state('');
  let pageNum = $state(1);
  let total = $state(0);

  let formOpen = $state(false);
  let secretOpen = $state(false);
  let secretKind = $state<'password' | 'pin'>('pin');
  let target = $state<Employee | null>(null);
  let busy = $state<string | null>(null);

  const columns = [
    { key: 'name', label: 'Employee' },
    { key: 'role', label: 'Role', width: '150px' },
    { key: 'branches', label: 'Branches' },
    { key: 'status', label: 'Status', width: '110px' },
    { key: 'last_login_at', label: 'Last sign-in', width: '130px' },
    { key: 'actions', label: '', width: '300px', align: 'right' as const },
  ];

  async function load() {
    loading = true;
    try {
      const res = await api.list<Employee>('/api/v1/t/employees', {
        page: pageNum,
        per_page: PER_PAGE,
        q,
        status,
      });
      rows = res.data;
      total = res.meta.total;
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Failed to load employees');
    } finally {
      loading = false;
    }
  }

  async function loadLookups() {
    try {
      [roles, branches] = await Promise.all([
        api.get<Ref[]>('/api/v1/t/employees/assignable-roles'),
        api.get<Branch[]>('/api/v1/t/branches').catch(() => [] as Branch[]),
      ]);
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Failed to load roles');
    }
  }

  function edit(e: Employee | null) {
    target = e;
    formOpen = true;
  }

  function secret(e: Employee, kind: 'password' | 'pin') {
    target = e;
    secretKind = kind;
    secretOpen = true;
  }

  async function setStatus(e: Employee, next: 'disable' | 'enable') {
    if (next === 'disable' && !confirm(`Disable ${e.name}? They will be signed out everywhere.`)) return;
    busy = e.id;
    try {
      await api.post(`/api/v1/t/employees/${e.id}/${next}`);
      toast.success(next === 'disable' ? 'Employee disabled' : 'Employee re-enabled');
      load();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : 'Update failed');
    } finally {
      busy = null;
    }
  }

  onMount(() => {
    load();
    loadLookups();
  });
</script>

<section class="head">
  <div>
    <h1>Employees</h1>
    <p class="muted">{total} employees</p>
  </div>
  <div class="tools">
    <TextField
      placeholder="Search name, email, username…"
      bind:value={q}
      oninput={() => {
        pageNum = 1;
        load();
      }}
    />
    <select
      bind:value={status}
      onchange={() => {
        pageNum = 1;
        load();
      }}
      aria-label="Status"
    >
      <option value="">All statuses</option>
      <option value="active">Active</option>
      <option value="disabled">Disabled</option>
    </select>
    {#if auth.can('users.employee.create') && roles.length > 0}
      <Button onclick={() => edit(null)}>Add employee</Button>
    {/if}
  </div>
</section>

<DataTable {columns} {rows} rowKey={(r) => r.id} {loading} empty="No employees match.">
  {#snippet cell(row: Employee, col)}
    {#if col.key === 'name'}
      <strong>{row.name}</strong>
      <div class="muted small">{row.email ?? row.username}</div>
    {:else if col.key === 'role'}
      {row.role.name}
    {:else if col.key === 'branches'}
      {#if row.all_branches}
        <Badge tone="info">All branches</Badge>
      {:else}
        {row.branches.map((b) => b.name).join(', ') || '—'}
      {/if}
    {:else if col.key === 'status'}
      <StatusDot status={row.status === 'active' ? 'ok' : 'error'} label={row.status} />
    {:else if col.key === 'last_login_at'}
      {row.last_login_at ? formatDate(row.last_login_at) : 'Never'}
    {:else if col.key === 'actions'}
      <div class="actions">
        {#if auth.can('users.employee.update')}
          <Button size="sm" variant="secondary" onclick={() => edit(row)}>Edit</Button>
        {/if}
        {#if row.id !== auth.user?.id}
          {#if auth.can('users.employee.reset_pin')}
            <Button size="sm" variant="ghost" onclick={() => secret(row, 'pin')}>PIN</Button>
          {/if}
          {#if auth.can('users.employee.reset_password')}
            <Button size="sm" variant="ghost" onclick={() => secret(row, 'password')}>Password</Button>
          {/if}
          {#if auth.can('users.employee.deactivate')}
            {#if row.status === 'active'}
              <Button
                size="sm"
                variant="danger"
                loading={busy === row.id}
                onclick={() => setStatus(row, 'disable')}
              >
                Disable
              </Button>
            {:else}
              <Button
                size="sm"
                variant="success"
                loading={busy === row.id}
                onclick={() => setStatus(row, 'enable')}
              >
                Enable
              </Button>
            {/if}
          {/if}
        {/if}
      </div>
    {/if}
  {/snippet}
</DataTable>

<Pager
  page={pageNum}
  perPage={PER_PAGE}
  {total}
  onpage={(p) => {
    pageNum = p;
    load();
  }}
/>

<EmployeeForm bind:open={formOpen} employee={target} {roles} {branches} onsaved={load} />
<SecretModal bind:open={secretOpen} employee={target} kind={secretKind} onsaved={load} />

<style>
  .head {
    display: flex;
    justify-content: space-between;
    align-items: end;
    gap: var(--space-4);
    margin-bottom: var(--space-5);
    flex-wrap: wrap;
  }
  .head h1 {
    margin: 0 0 var(--space-1);
  }
  .tools {
    display: flex;
    gap: var(--space-3);
    align-items: center;
    flex-wrap: wrap;
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
    flex-wrap: wrap;
  }
</style>
