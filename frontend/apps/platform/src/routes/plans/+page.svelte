<script lang="ts">
  import { onMount } from 'svelte';
  import { ApiError } from '@mauzo/api-client';
  import type { Plan } from '@mauzo/types';
  import { Button, DataTable, Modal, TextField, toast } from '@mauzo/ui';
  import { api } from '#lib/api.ts';

  let rows = $state<Plan[]>([]);
  let loading = $state(true);

  let open = $state(false);
  let editTarget = $state<Plan | null>(null);
  let saving = $state(false);
  let formError = $state<string | null>(null);
  let form = $state({
    name: '',
    description: '',
    price_kes: '0',
    billing_interval: 'monthly',
    modules: '',
    max_branches: '',
    max_users: '',
    is_active: true,
    sort: '0',
  });

  const columns = [
    { key: 'name', label: 'Name' },
    { key: 'price_kes', label: 'Price', width: '110px' },
    { key: 'billing_interval', label: 'Billing', width: '90px' },
    { key: 'limits', label: 'Limits', width: '140px' },
    { key: 'is_active', label: 'Status', width: '90px' },
    { key: 'actions', label: '', width: '90px', align: 'right' as const },
  ];

  async function load() {
    loading = true;
    try {
      rows = (await api.get<Plan[]>('/api/v1/platform/plans')) ?? [];
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Failed to load plans');
    } finally {
      loading = false;
    }
  }

  function openCreate() {
    editTarget = null;
    form = {
      name: '',
      description: '',
      price_kes: '0',
      billing_interval: 'monthly',
      modules: '',
      max_branches: '',
      max_users: '',
      is_active: true,
      sort: '0',
    };
    formError = null;
    open = true;
  }

  function openEdit(p: Plan) {
    editTarget = p;
    form = {
      name: p.name,
      description: p.description ?? '',
      price_kes: p.price_kes,
      billing_interval: p.billing_interval,
      modules: p.modules.join(', '),
      max_branches: p.max_branches?.toString() ?? '',
      max_users: p.max_users?.toString() ?? '',
      is_active: p.is_active,
      sort: String(p.sort),
    };
    formError = null;
    open = true;
  }

  function body() {
    return {
      name: form.name.trim(),
      description: form.description.trim(),
      price_kes: form.price_kes.trim(),
      billing_interval: form.billing_interval,
      modules: form.modules
        .split(',')
        .map((m) => m.trim())
        .filter(Boolean),
      max_branches: form.max_branches === '' ? null : parseInt(form.max_branches, 10),
      max_users: form.max_users === '' ? null : parseInt(form.max_users, 10),
      is_active: form.is_active,
      sort: parseInt(form.sort, 10) || 0,
    };
  }

  async function save() {
    saving = true;
    formError = null;
    try {
      if (editTarget) {
        await api.patch(`/api/v1/platform/plans/${editTarget.id}`, body());
        toast.success('Plan updated');
      } else {
        await api.post('/api/v1/platform/plans', body());
        toast.success('Plan created');
      }
      open = false;
      load();
    } catch (e) {
      formError = e instanceof ApiError ? e.message : 'Save failed';
    } finally {
      saving = false;
    }
  }

  onMount(load);
</script>

<div class="page">
  <header>
    <h1>Plans</h1>
    <Button onclick={openCreate}>New plan</Button>
  </header>

  <DataTable {columns} {rows} rowKey={(p) => p.id} {loading} empty="No plans yet.">
    {#snippet cell(p: Plan, col)}
      {#if col.key === 'name'}
        <strong>{p.name}</strong>{#if p.description}<div class="muted">{p.description}</div>{/if}
      {:else if col.key === 'price_kes'}
        KES {p.price_kes}
      {:else if col.key === 'limits'}
        <span class="muted">{p.max_branches ?? '∞'} branches · {p.max_users ?? '∞'} users</span>
      {:else if col.key === 'is_active'}
        {p.is_active ? 'active' : 'hidden'}
      {:else if col.key === 'actions'}
        <Button size="sm" variant="secondary" onclick={() => openEdit(p)}>Edit</Button>
      {:else}
        {(p as unknown as Record<string, unknown>)[col.key]}
      {/if}
    {/snippet}
  </DataTable>
</div>

<Modal bind:open title={editTarget ? 'Edit plan' : 'New plan'} locked={saving}>
  <form
    class="form"
    onsubmit={(e) => {
      e.preventDefault();
      save();
    }}
  >
    {#if formError}<p class="error">{formError}</p>{/if}
    <TextField label="Name" bind:value={form.name} required />
    <TextField label="Description" bind:value={form.description} />
    <div class="grid2">
      <TextField label="Price (KES)" bind:value={form.price_kes} required />
      <label class="field">
        <span>Billing</span>
        <select bind:value={form.billing_interval}>
          <option value="monthly">monthly</option>
          <option value="yearly">yearly</option>
        </select>
      </label>
    </div>
    <label class="field">
      <span>Modules (comma-separated)</span>
      <textarea rows="3" bind:value={form.modules}></textarea>
    </label>
    <div class="grid2">
      <TextField label="Max branches (blank = ∞)" bind:value={form.max_branches} />
      <TextField label="Max users (blank = ∞)" bind:value={form.max_users} />
    </div>
    <div class="grid2">
      <TextField label="Sort order" bind:value={form.sort} />
      <label class="check">
        <input type="checkbox" bind:checked={form.is_active} /> Active (visible to tenants)
      </label>
    </div>
  </form>
  {#snippet footer()}
    <Button variant="secondary" onclick={() => (open = false)}>Cancel</Button>
    <Button loading={saving} onclick={save}>Save</Button>
  {/snippet}
</Modal>

<style>
  .page {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }
  header {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }
  h1 {
    margin: 0;
    font-size: 1.4rem;
  }
  .muted {
    color: var(--muted, #6b7280);
    font-size: 0.85rem;
  }
  .form {
    display: flex;
    flex-direction: column;
    gap: 0.9rem;
  }
  .grid2 {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 0.75rem;
  }
  .field {
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
    font-size: 0.9rem;
  }
  .field select,
  .field textarea {
    padding: 0.5rem 0.6rem;
    border: 1px solid var(--border, #d1d5db);
    border-radius: 6px;
    font: inherit;
    background: var(--surface, #fff);
    color: inherit;
  }
  .check {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.9rem;
    margin-top: 1.4rem;
  }
  .error {
    color: var(--danger, #dc2626);
    margin: 0;
    font-size: 0.9rem;
  }
</style>
