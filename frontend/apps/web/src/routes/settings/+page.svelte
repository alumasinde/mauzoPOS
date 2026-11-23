<script lang="ts">
  import { onMount } from 'svelte';
  import { ApiError } from '@mauzo/api-client';
  import type { SettingEntry } from '@mauzo/types';
  import { Button, DataTable, Modal, toast, PageHeader } from '@mauzo/ui';
  import { api } from '#lib/api.ts';

  let rows = $state<SettingEntry[]>([]);
  let loading = $state(true);

  let open = $state(false);
  let editKey = $state('');
  let editValue = $state('');
  let saving = $state(false);
  let formError = $state<string | null>(null);

  const columns = [
    { key: 'key', label: 'Key' },
    { key: 'value', label: 'Value' },
    { key: 'updated_at', label: 'Updated', width: '170px' },
    { key: 'actions', label: '', width: '80px', align: 'right' as const },
  ];

  async function load() {
    loading = true;
    try {
      rows = (await api.get<SettingEntry[]>('/api/v1/t/settings')) ?? [];
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Failed to load settings');
    } finally {
      loading = false;
    }
  }

  function openEdit(e: SettingEntry) {
    editKey = e.key;
    editValue = JSON.stringify(e.value, null, 2);
    formError = null;
    open = true;
  }

  async function save() {
    saving = true;
    formError = null;
    try {
      let value: unknown;
      try {
        value = JSON.parse(editValue);
      } catch {
        formError = 'Value must be valid JSON (e.g. true, "text", 16, ["a","b"])';
        return;
      }
      await api.put(`/api/v1/t/settings/${editKey}`, { value });
      toast.success('Setting saved');
      open = false;
      load();
    } catch (e) {
      formError = e instanceof ApiError ? e.message : 'Save failed';
    } finally {
      saving = false;
    }
  }

  function fmt(e: SettingEntry) {
    const s = JSON.stringify(e.value);
    return s.length > 80 ? s.slice(0, 80) + '…' : s;
  }

  onMount(load);
</script>

<div class="page">
  <PageHeader title="Settings" sub="Business preferences, tax and receipt options." icon="cog" />
  <DataTable
    {columns}
    {rows}
    rowKey={(e) => e.key}
    {loading}
    empty="No settings yet — run onboarding to apply profile defaults."
  >
    {#snippet cell(e: SettingEntry, col)}
      {#if col.key === 'key'}
        <code>{e.key}</code>
      {:else if col.key === 'value'}
        <span class="muted"><code>{fmt(e)}</code></span>
      {:else if col.key === 'updated_at'}
        <span class="muted">{new Date(e.updated_at).toLocaleString()}</span>
      {:else if col.key === 'actions'}
        <Button size="sm" variant="secondary" onclick={() => openEdit(e)}>Edit</Button>
      {/if}
    {/snippet}
  </DataTable>
</div>

<Modal bind:open title="Edit setting" locked={saving}>
  <form
    class="form"
    onsubmit={(e) => {
      e.preventDefault();
      save();
    }}
  >
    {#if formError}<p class="error">{formError}</p>{/if}
    <p class="muted"><code>{editKey}</code></p>
    <label class="field">
      <span>Value (JSON)</span>
      <textarea rows="5" bind:value={editValue}></textarea>
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
  .muted {
    color: var(--muted, #6b7280);
    font-size: 0.85rem;
    margin: 0;
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
  .field textarea {
    padding: 0.5rem 0.6rem;
    border: 1px solid var(--border, #d1d5db);
    border-radius: 6px;
    font: inherit;
    font-family: monospace;
    background: var(--surface, #fff);
    color: inherit;
  }
  .error {
    color: var(--danger, #dc2626);
    margin: 0;
    font-size: 0.9rem;
  }
</style>
