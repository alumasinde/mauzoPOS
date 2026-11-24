<script lang="ts">
  // Branches — the physical locations of the business. Create, edit,
  // mark the main branch and close locations.
  import { onMount } from 'svelte';
  import { ApiError } from '@mauzo/api-client';
  import { Badge, Button, DataTable, Icon, Modal, PageHeader, TextField, toast } from '@mauzo/ui';
  import type { Branch } from '@mauzo/types';
  import { api } from '#lib/api.ts';
  import { auth } from '#lib/auth.svelte.ts';
  import { branch } from '#lib/branch.svelte.ts';

  let rows = $state<Branch[]>([]);
  let loading = $state(true);
  let busy = $state(false);
  let editing = $state<Branch | 'new' | null>(null);
  let form = $state({ code: '', name: '', phone: '', location: '', is_main: false });

  const canManage = $derived(auth.can('branches.branch.manage'));

  const columns = [
    { key: 'code', label: 'Code', width: '90px' },
    { key: 'name', label: 'Name' },
    { key: 'phone', label: 'Phone', width: '150px' },
    { key: 'location', label: 'Location' },
    { key: 'main', label: 'Main', width: '80px' },
    { key: 'status', label: 'Status', width: '100px' },
    { key: 'actions', label: '', width: '260px', align: 'right' as const },
  ];

  async function load() {
    loading = true;
    try {
      rows = await api.get<Branch[]>('/api/v1/t/branches');
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Failed to load branches');
    } finally {
      loading = false;
    }
  }

  function openForm(b: Branch | 'new') {
    editing = b;
    form =
      b === 'new'
        ? { code: '', name: '', phone: '', location: '', is_main: false }
        : {
            code: b.code,
            name: b.name,
            phone: b.phone ?? '',
            location: b.location ?? '',
            is_main: b.is_main,
          };
  }

  async function save() {
    if (!editing || !form.name.trim()) return;
    busy = true;
    try {
      if (editing === 'new') {
        await api.post('/api/v1/t/branches', { ...form, code: form.code.trim() });
        toast.success('Branch created');
      } else {
        await api.patch(`/api/v1/t/branches/${editing.id}`, form);
        toast.success('Branch updated');
      }
      editing = null;
      await Promise.all([load(), branch.load(auth.user?.all_branches ?? false)]);
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Could not save the branch');
    } finally {
      busy = false;
    }
  }

  async function setStatus(b: Branch, status: 'active' | 'closed') {
    busy = true;
    try {
      await api.patch(`/api/v1/t/branches/${b.id}`, { status });
      toast.success(status === 'closed' ? `${b.name} closed` : `${b.name} reopened`);
      await load();
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Could not update the branch');
    } finally {
      busy = false;
    }
  }

  onMount(load);
</script>

<PageHeader
  title="Branches"
  icon="building"
  sub="The physical locations of the business — the main branch is the default for the POS and the back office."
>
  {#snippet actions()}
    {#if canManage}
      <Button variant="primary" onclick={() => openForm('new')}>
        <Icon name="plus" size={16} /> New branch
      </Button>
    {/if}
  {/snippet}
</PageHeader>

<DataTable {columns} {rows} rowKey={(b) => b.id} {loading} empty="No branches yet — add the first location.">
  {#snippet cell(b: Branch, col)}
    {#if col.key === 'code'}<strong>{b.code}</strong>
    {:else if col.key === 'name'}{b.name}
    {:else if col.key === 'phone'}{b.phone || '—'}
    {:else if col.key === 'location'}{b.location || '—'}
    {:else if col.key === 'main'}
      {#if b.is_main}<Badge tone="info">Main</Badge>{:else}—{/if}
    {:else if col.key === 'status'}
      <Badge tone={b.status === 'active' ? 'success' : 'neutral'}>{b.status}</Badge>
    {:else if col.key === 'actions'}
      <span class="flex gap-2 justify-end">
        {#if canManage}
          <Button size="sm" variant="ghost" onclick={() => openForm(b)}>Edit</Button>
          {#if b.status === 'active'}
            <Button size="sm" variant="ghost" loading={busy} onclick={() => setStatus(b, 'closed')}>
              Close
            </Button>
          {:else}
            <Button size="sm" variant="ghost" loading={busy} onclick={() => setStatus(b, 'active')}>
              Reopen
            </Button>
          {/if}
        {/if}
      </span>
    {/if}
  {/snippet}
</DataTable>

<Modal
  open={editing !== null}
  title={editing === 'new' ? 'New branch' : 'Edit branch'}
  onclose={() => (editing = null)}
>
  <div class="space-y-3">
    <TextField label="Code" bind:value={form.code} placeholder="e.g. MAIN" />
    <TextField label="Name" bind:value={form.name} placeholder="Branch name" />
    <TextField label="Phone" bind:value={form.phone} placeholder="Optional" />
    <TextField label="Location" bind:value={form.location} placeholder="Address / city" />
    <label class="flex items-center gap-2 text-sm">
      <input type="checkbox" bind:checked={form.is_main} />
      Main branch (default for the POS and back office)
    </label>
  </div>
  {#snippet footer()}
    <Button variant="ghost" onclick={() => (editing = null)}>Cancel</Button>
    <Button variant="primary" loading={busy} onclick={save}>Save</Button>
  {/snippet}
</Modal>
