<script lang="ts">
  import { onMount } from 'svelte';
  import { ApiError } from '@mauzo/api-client';
  import type { BusinessProfile, BusinessType } from '@mauzo/types';
  import { Button, DataTable, Modal, TextField, toast } from '@mauzo/ui';
  import { api } from '#lib/api.ts';

  let types = $state<BusinessType[]>([]);
  let profiles = $state<BusinessProfile[]>([]);
  let selected = $state<string>('');
  let loading = $state(true);

  let open = $state(false);
  let editTarget = $state<BusinessProfile | null>(null);
  let saving = $state(false);
  let formError = $state<string | null>(null);
  let newType = $state('');
  let form = $state({ name: '', is_default: false, modules: '', categories: '', settings: '{}' });

  const columns = [
    { key: 'name', label: 'Profile' },
    { key: 'business_type', label: 'Type', width: '120px' },
    { key: 'modules', label: 'Modules', width: '90px' },
    { key: 'is_default', label: 'Default', width: '80px' },
    { key: 'actions', label: '', width: '90px', align: 'right' as const },
  ];

  const shown = $derived(profiles.filter((p) => selected === '' || p.business_type === selected));

  async function load() {
    loading = true;
    try {
      const [t, p] = await Promise.all([
        api.get<BusinessType[]>('/api/v1/platform/business-types'),
        api.get<BusinessProfile[]>('/api/v1/platform/business-profiles'),
      ]);
      types = t ?? [];
      profiles = p ?? [];
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Failed to load');
    } finally {
      loading = false;
    }
  }

  function openCreate() {
    editTarget = null;
    form = { name: '', is_default: false, modules: '', categories: '', settings: '{}' };
    formError = null;
    open = true;
  }

  function openEdit(p: BusinessProfile) {
    editTarget = p;
    form = {
      name: p.name,
      is_default: p.is_default,
      modules: p.definition.modules.join(', '),
      categories: p.definition.categories.join(', '),
      settings: JSON.stringify(p.definition.settings ?? {}, null, 2),
    };
    formError = null;
    open = true;
  }

  function body() {
    let settings: Record<string, unknown>;
    try {
      settings = JSON.parse(form.settings) as Record<string, unknown>;
    } catch {
      throw new Error('Settings must be valid JSON');
    }
    return {
      business_type: editTarget?.business_type ?? newType,
      name: form.name.trim(),
      is_default: form.is_default,
      definition: {
        modules: form.modules
          .split(',')
          .map((m) => m.trim())
          .filter(Boolean),
        categories: form.categories
          .split(',')
          .map((c) => c.trim())
          .filter(Boolean),
        settings,
        receipt_template: editTarget?.definition.receipt_template ?? '',
        pos_layout: editTarget?.definition.pos_layout ?? {},
      },
    };
  }

  async function save() {
    saving = true;
    formError = null;
    try {
      const payload = body();
      if (editTarget) {
        await api.patch(`/api/v1/platform/business-profiles/${editTarget.id}`, payload);
        toast.success('Profile updated');
      } else {
        await api.post('/api/v1/platform/business-profiles', payload);
        toast.success('Profile created');
      }
      open = false;
      load();
    } catch (e) {
      formError = e instanceof Error ? e.message : 'Save failed';
    } finally {
      saving = false;
    }
  }

  onMount(load);
</script>

<div class="page">
  <header>
    <h1>Business types &amp; profiles</h1>
    <Button onclick={openCreate}>New profile</Button>
  </header>

  <div class="typebar">
    <button class:active={selected === ''} onclick={() => (selected = '')}>All</button>
    {#each types as t (t.code)}
      <button class:active={selected === t.code} onclick={() => (selected = t.code)}>{t.name}</button>
    {/each}
  </div>

  <DataTable {columns} rows={shown} rowKey={(p) => p.id} {loading} empty="No profiles.">
    {#snippet cell(p: BusinessProfile, col)}
      {#if col.key === 'name'}
        <strong>{p.name}</strong>
      {:else if col.key === 'modules'}
        <span class="muted">{p.definition.modules.length}</span>
      {:else if col.key === 'is_default'}
        {p.is_default ? 'default' : ''}
      {:else if col.key === 'actions'}
        <Button size="sm" variant="secondary" onclick={() => openEdit(p)}>Edit</Button>
      {:else}
        {(p as unknown as Record<string, unknown>)[col.key]}
      {/if}
    {/snippet}
  </DataTable>
</div>

<Modal bind:open title={editTarget ? 'Edit profile' : 'New profile'} locked={saving}>
  <form
    class="form"
    onsubmit={(e) => {
      e.preventDefault();
      save();
    }}
  >
    {#if formError}<p class="error">{formError}</p>{/if}
    {#if !editTarget}
      <label class="field">
        <span>Business type</span>
        <select bind:value={newType} required>
          <option value="" disabled>Choose…</option>
          {#each types as t (t.code)}<option value={t.code}>{t.name}</option>{/each}
        </select>
      </label>
    {/if}
    <TextField label="Name" bind:value={form.name} required />
    <label class="field">
      <span>Modules (comma-separated)</span>
      <textarea rows="3" bind:value={form.modules}></textarea>
    </label>
    <label class="field">
      <span>Default categories</span>
      <textarea rows="2" bind:value={form.categories}></textarea>
    </label>
    <label class="field">
      <span>Settings defaults (JSON)</span>
      <textarea rows="4" bind:value={form.settings}></textarea>
    </label>
    <label class="check">
      <input type="checkbox" bind:checked={form.is_default} /> Default profile for this type
    </label>
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
  .typebar {
    display: flex;
    gap: 0.4rem;
    flex-wrap: wrap;
  }
  .typebar button {
    padding: 0.35rem 0.8rem;
    border: 1px solid var(--border, #d1d5db);
    border-radius: 999px;
    background: transparent;
    cursor: pointer;
    font: inherit;
    color: inherit;
  }
  .typebar button.active {
    background: var(--primary, #2563eb);
    color: #fff;
    border-color: transparent;
  }
  .muted {
    color: var(--muted, #6b7280);
  }
  .form {
    display: flex;
    flex-direction: column;
    gap: 0.9rem;
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
  }
  .error {
    color: var(--danger, #dc2626);
    margin: 0;
    font-size: 0.9rem;
  }
</style>
