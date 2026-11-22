<script lang="ts">
  import { onMount } from 'svelte';
  import { ApiError } from '@mauzo/api-client';
  import type { PlatformUser } from '@mauzo/types';
  import { Button, DataTable, Modal, StatusDot, TextField, toast } from '@mauzo/ui';
  import { api } from '#lib/api.ts';
  import { auth } from '#lib/auth.ts';

  let rows = $state<PlatformUser[]>([]);
  let loading = $state(true);
  let q = $state('');

  let createOpen = $state(false);
  let pwOpen = $state(false);
  let pwTarget = $state<PlatformUser | null>(null);
  let newPassword = $state('');
  let form = $state({ name: '', email: '', password: '', role: 'support' });
  let saving = $state(false);
  let formError = $state<string | null>(null);

  const columns = [
    { key: 'name', label: 'Name' },
    { key: 'email', label: 'Email' },
    { key: 'role', label: 'Role', width: '120px' },
    { key: 'mfa', label: 'MFA', width: '70px' },
    { key: 'status', label: 'Status', width: '100px' },
    { key: 'actions', label: '', width: '240px', align: 'right' as const },
  ];

  async function load() {
    loading = true;
    try {
      const res = await api.list<PlatformUser>('/api/v1/platform/users', { q });
      rows = res.data;
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Failed to load users');
    } finally {
      loading = false;
    }
  }

  async function create() {
    saving = true;
    formError = null;
    try {
      await api.post('/api/v1/platform/users', {
        name: form.name.trim(),
        email: form.email.trim(),
        password: form.password,
        role: form.role,
      });
      createOpen = false;
      form = { name: '', email: '', password: '', role: 'support' };
      toast.success('User created');
      load();
    } catch (e) {
      formError = e instanceof ApiError ? e.message : 'Create failed';
    } finally {
      saving = false;
    }
  }

  async function setStatus(u: PlatformUser, action: 'enable' | 'disable') {
    try {
      await api.post(`/api/v1/platform/users/${u.id}/${action}`);
      toast.success(`User ${action}d`);
      load();
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Action failed');
    }
  }

  async function resetPassword() {
    saving = true;
    formError = null;
    try {
      await api.post(`/api/v1/platform/users/${pwTarget!.id}/password`, { password: newPassword });
      pwOpen = false;
      newPassword = '';
      toast.success('Password updated');
    } catch (e) {
      formError = e instanceof ApiError ? e.message : 'Reset failed';
    } finally {
      saving = false;
    }
  }

  onMount(load);
</script>

<section class="head">
  <div>
    <h1>Platform users</h1>
    <p class="muted">SaaS operator accounts</p>
  </div>
  <div class="tools">
    <TextField placeholder="Search…" bind:value={q} oninput={load} />
    <Button onclick={() => (createOpen = true)}>New user</Button>
  </div>
</section>

<DataTable {columns} {rows} rowKey={(r) => r.id} {loading} empty="No users.">
  {#snippet cell(row: PlatformUser, col)}
    {#if col.key === 'role'}
      {row.role.replace('_', ' ')}
    {:else if col.key === 'mfa'}
      {row.mfa_enabled ? 'on' : '—'}
    {:else if col.key === 'status'}
      <StatusDot status={row.status === 'active' ? 'ok' : 'error'} label={row.status} />
    {:else if col.key === 'actions'}
      <Button
        size="sm"
        variant="secondary"
        onclick={() => {
          pwTarget = row;
          pwOpen = true;
        }}
      >
        Password
      </Button>
      {#if row.id !== auth.user?.id}
        {#if row.status === 'active'}
          <Button size="sm" variant="danger" onclick={() => setStatus(row, 'disable')}>Disable</Button>
        {:else}
          <Button size="sm" variant="success" onclick={() => setStatus(row, 'enable')}>Enable</Button>
        {/if}
      {/if}
    {:else}
      {(row as unknown as Record<string, string>)[col.key]}
    {/if}
  {/snippet}
</DataTable>

<Modal bind:open={createOpen} title="New platform user" locked={saving}>
  <form
    class="form"
    onsubmit={(e) => {
      e.preventDefault();
      create();
    }}
  >
    <TextField label="Name" bind:value={form.name} required />
    <TextField label="Email" type="email" bind:value={form.email} required />
    <TextField
      label="Temporary password"
      type="password"
      bind:value={form.password}
      required
      hint="At least 8 characters"
    />
    <label class="field">
      <span>Role</span>
      <select bind:value={form.role}>
        <option value="super_admin">Super admin</option>
        <option value="support">Support</option>
        <option value="billing">Billing</option>
        <option value="viewer">Viewer</option>
      </select>
    </label>
    {#if formError}<p class="error">{formError}</p>{/if}
  </form>
  {#snippet footer()}
    <Button variant="secondary" onclick={() => (createOpen = false)}>Cancel</Button>
    <Button onclick={create} loading={saving}>Create</Button>
  {/snippet}
</Modal>

<Modal bind:open={pwOpen} title="Set password — {pwTarget?.email ?? ''}" locked={saving}>
  <form
    class="form"
    onsubmit={(e) => {
      e.preventDefault();
      resetPassword();
    }}
  >
    <TextField
      label="New password"
      type="password"
      bind:value={newPassword}
      required
      hint="At least 8 characters"
    />
    {#if formError}<p class="error">{formError}</p>{/if}
  </form>
  {#snippet footer()}
    <Button variant="secondary" onclick={() => (pwOpen = false)}>Cancel</Button>
    <Button onclick={resetPassword} loading={saving}>Set password</Button>
  {/snippet}
</Modal>

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
  .form {
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
  }
  .field {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
    font-size: var(--text-sm);
    font-weight: 600;
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
  .error {
    color: var(--color-danger);
    margin: 0;
    font-size: var(--text-sm);
  }
</style>
