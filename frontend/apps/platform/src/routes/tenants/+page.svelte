<script lang="ts">
  import { onMount } from 'svelte';
  import { ApiError } from '@mauzo/api-client';
  import type { Tenant } from '@mauzo/types';
  import { Button, DataTable, Modal, StatusDot, TextField, toast } from '@mauzo/ui';
  import { formatDate } from '@mauzo/utils';
  import { api } from '#lib/api.ts';

  let rows = $state<Tenant[]>([]);
  let loading = $state(true);
  let q = $state('');
  let pageNum = $state(1);
  let total = $state(0);

  let createOpen = $state(false);
  let form = $state({ name: '', slug: '', email: '', phone: '' });
  let saving = $state(false);
  let formError = $state<string | null>(null);

  let editOpen = $state(false);
  let editTarget = $state<Tenant | null>(null);
  let editForm = $state({ name: '', email: '', phone: '' });

  const columns = [
    { key: 'name', label: 'Name' },
    { key: 'slug', label: 'Sign-in code' },
    { key: 'status', label: 'Status', width: '110px' },
    { key: 'created_at', label: 'Created', width: '130px' },
    { key: 'actions', label: '', width: '220px', align: 'right' as const },
  ];

  async function load() {
    loading = true;
    try {
      const res = await api.list<Tenant>('/api/v1/platform/tenants', { page: pageNum, q });
      rows = res.data;
      total = res.meta.total;
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Failed to load tenants');
    } finally {
      loading = false;
    }
  }

  async function create() {
    saving = true;
    formError = null;
    try {
      await api.post('/api/v1/platform/tenants', {
        name: form.name.trim(),
        slug: form.slug.trim(),
        email: form.email.trim(),
        phone: form.phone.trim(),
      });
      createOpen = false;
      form = { name: '', slug: '', email: '', phone: '' };
      toast.success('Tenant created');
      load();
    } catch (e) {
      formError = e instanceof ApiError ? e.message : 'Create failed';
    } finally {
      saving = false;
    }
  }

  function openEdit(t: Tenant) {
    editTarget = t;
    editForm = { name: t.name, email: t.email ?? '', phone: t.phone ?? '' };
    formError = null;
    editOpen = true;
  }

  async function saveEdit() {
    if (!editTarget) return;
    saving = true;
    formError = null;
    try {
      await api.patch(`/api/v1/platform/tenants/${editTarget.id}`, {
        name: editForm.name.trim(),
        email: editForm.email.trim(),
        phone: editForm.phone.trim(),
      });
      editOpen = false;
      toast.success('Tenant updated');
      load();
    } catch (e) {
      formError = e instanceof ApiError ? e.message : 'Update failed';
    } finally {
      saving = false;
    }
  }

  async function setStatus(t: Tenant, action: 'suspend' | 'reactivate') {
    const reason = action === 'suspend' ? (prompt('Suspension reason (optional):') ?? '') : '';
    try {
      await api.post(`/api/v1/platform/tenants/${t.id}/${action}`, { reason });
      toast.success(action === 'suspend' ? 'Tenant suspended' : 'Tenant reactivated');
      load();
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Action failed');
    }
  }

  onMount(load);
</script>

<section class="head">
  <div>
    <h1>Tenants</h1>
    <p class="muted">{total} businesses</p>
  </div>
  <div class="tools">
    <TextField
      placeholder="Search…"
      bind:value={q}
      oninput={() => {
        pageNum = 1;
        load();
      }}
    />
    <Button onclick={() => (createOpen = true)}>New tenant</Button>
  </div>
</section>

<DataTable {columns} {rows} rowKey={(r) => r.id} {loading} empty="No tenants yet.">
  {#snippet cell(row: Tenant, col)}
    {#if col.key === 'status'}
      <StatusDot status={row.status === 'active' ? 'ok' : 'error'} label={row.status} />
    {:else if col.key === 'created_at'}
      {formatDate(row.created_at)}
    {:else if col.key === 'actions'}
      <Button size="sm" variant="secondary" onclick={() => openEdit(row)}>Edit</Button>
      {#if row.status === 'active'}
        <Button size="sm" variant="danger" onclick={() => setStatus(row, 'suspend')}>Suspend</Button>
      {:else}
        <Button size="sm" variant="success" onclick={() => setStatus(row, 'reactivate')}>Reactivate</Button>
      {/if}
    {:else}
      {(row as unknown as Record<string, string>)[col.key]}
    {/if}
  {/snippet}
</DataTable>

<Modal bind:open={createOpen} title="New tenant" locked={saving}>
  <form
    class="form"
    onsubmit={(e) => {
      e.preventDefault();
      create();
    }}
  >
    <TextField label="Business name" bind:value={form.name} required />
    <TextField
      label="Sign-in code (slug)"
      bind:value={form.slug}
      required
      hint="lowercase, digits, hyphens — staff use it to sign in"
    />
    <TextField label="Email" type="email" bind:value={form.email} />
    <TextField label="Phone" bind:value={form.phone} />
    {#if formError}<p class="error">{formError}</p>{/if}
  </form>
  {#snippet footer()}
    <Button variant="secondary" onclick={() => (createOpen = false)}>Cancel</Button>
    <Button onclick={create} loading={saving}>Create</Button>
  {/snippet}
</Modal>

<Modal bind:open={editOpen} title="Edit — {editTarget?.name ?? ''}" locked={saving}>
  <form
    class="form"
    onsubmit={(e) => {
      e.preventDefault();
      saveEdit();
    }}
  >
    <TextField label="Business name" bind:value={editForm.name} required />
    <TextField label="Sign-in code (slug)" value={editTarget?.slug ?? ''} disabled hint="cannot be changed" />
    <TextField label="Email" type="email" bind:value={editForm.email} />
    <TextField label="Phone" bind:value={editForm.phone} />
    {#if formError}<p class="error">{formError}</p>{/if}
  </form>
  {#snippet footer()}
    <Button variant="secondary" onclick={() => (editOpen = false)}>Cancel</Button>
    <Button onclick={saveEdit} loading={saving}>Save</Button>
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
  .error {
    color: var(--color-danger);
    margin: 0;
    font-size: var(--text-sm);
  }
</style>
