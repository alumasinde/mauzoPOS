<script lang="ts">
  import { onMount } from 'svelte';
  import { ApiError } from '@mauzo/api-client';
  import type { Branch, IssuedTerminalCode, PosTerminal } from '@mauzo/types';
  import { Button, DataTable, Icon, Modal, PageHeader, TextField, toast } from '@mauzo/ui';
  import { formatDate } from '@mauzo/utils';
  import { api } from '#lib/api.ts';
  import { auth } from '#lib/auth.svelte.ts';
  import CodeCard from '#lib/terminals/CodeCard.svelte';
  import RegisterModal from '#lib/terminals/RegisterModal.svelte';
  import TerminalState from '#lib/terminals/TerminalState.svelte';

  let rows = $state<PosTerminal[]>([]);
  let branches = $state<Branch[]>([]);
  let loading = $state(true);
  let busy = $state<string | null>(null);

  let registerOpen = $state(false);
  let codeOpen = $state(false);
  let codeFor = $state('');
  let issued = $state<IssuedTerminalCode | null>(null);

  let renameFor = $state<PosTerminal | null>(null);
  let newName = $state('');

  const canManage = $derived(auth.can('terminals.terminal.manage'));
  const canCode = $derived(auth.can('terminals.terminal.activate'));

  const columns = [
    { key: 'name', label: 'Till' },
    { key: 'branch', label: 'Branch' },
    { key: 'status', label: 'Status', width: '260px' },
    { key: 'device', label: 'Computer' },
    { key: 'seen', label: 'Last seen', width: '160px' },
    { key: 'actions', label: '', width: '300px', align: 'right' as const },
  ];

  async function load() {
    loading = true;
    try {
      [rows, branches] = await Promise.all([
        api.get<PosTerminal[]>('/api/v1/t/terminals'),
        api.get<Branch[]>('/api/v1/t/branches/accessible').catch(() => [] as Branch[]),
      ]);
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Failed to load tills');
    } finally {
      loading = false;
    }
  }

  function showCode(name: string, code: IssuedTerminalCode) {
    codeFor = name;
    issued = code;
    codeOpen = true;
  }

  async function getCode(t: PosTerminal) {
    if (
      t.status === 'active' &&
      !confirm(
        `${t.name} is paired with "${t.device_name ?? 'a computer'}". A new code lets you move it to another computer — the current one will be signed out when the code is used. Continue?`,
      )
    )
      return;
    busy = t.id;
    try {
      showCode(t.name, await api.post<IssuedTerminalCode>(`/api/v1/t/terminals/${t.id}/code`));
      await load();
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Could not get a code');
    } finally {
      busy = null;
    }
  }

  async function patch(t: PosTerminal, body: Record<string, string>, done: string) {
    busy = t.id;
    try {
      await api.patch(`/api/v1/t/terminals/${t.id}`, body);
      toast.success(done);
      await load();
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Could not update the till');
    } finally {
      busy = null;
    }
  }

  function disable(t: PosTerminal) {
    if (
      confirm(
        `Disable ${t.name}? It is signed out immediately and cannot sell until enabled and paired again.`,
      )
    )
      void patch(t, { status: 'disabled' }, `${t.name} disabled`);
  }

  async function rename() {
    if (!renameFor || !newName.trim()) return;
    await patch(renameFor, { name: newName.trim() }, 'Renamed');
    renameFor = null;
  }

  onMount(load);
</script>

<PageHeader
  title="POS terminals"
  sub="Each till is paired to one computer with a one-time code. Codes last 24 hours."
  icon="monitor"
>
  {#snippet actions()}
    {#if canManage}
      <Button onclick={() => (registerOpen = true)} disabled={!branches.length}>
        <Icon name="plus" size={16} /> Add till
      </Button>
    {/if}
  {/snippet}
</PageHeader>

<DataTable
  {columns}
  {rows}
  rowKey={(t) => t.id}
  {loading}
  empty="No tills yet — add one to get its activation code."
>
  {#snippet cell(t: PosTerminal, col)}
    {#if col.key === 'name'}<strong>{t.name}</strong>
    {:else if col.key === 'branch'}{t.branch ?? '—'}
    {:else if col.key === 'status'}<TerminalState {t} />
    {:else if col.key === 'device'}{t.device_name ?? '—'}
    {:else if col.key === 'seen'}{t.last_seen_at ? formatDate(t.last_seen_at) : '—'}
    {:else if col.key === 'actions'}
      <span class="actions">
        {#if canCode && t.status !== 'disabled'}
          <Button
            size="sm"
            variant={t.status === 'pending' ? 'primary' : 'secondary'}
            loading={busy === t.id}
            onclick={() => getCode(t)}
          >
            {t.status === 'pending' ? 'Get activation code' : 'Re-pair'}
          </Button>
        {/if}
        {#if canManage}
          <Button size="sm" variant="ghost" onclick={() => ((renameFor = t), (newName = t.name))}>
            Rename
          </Button>
          {#if t.status === 'disabled'}
            <Button
              size="sm"
              variant="ghost"
              onclick={() => patch(t, { status: 'enabled' }, `${t.name} enabled — get a code to pair it`)}
            >
              Enable
            </Button>
          {:else}
            <Button size="sm" variant="ghost" onclick={() => disable(t)}>Disable</Button>
          {/if}
        {/if}
      </span>
    {/if}
  {/snippet}
</DataTable>

<RegisterModal
  bind:open={registerOpen}
  {branches}
  existing={rows.length}
  oncreated={(t, code) => {
    showCode(t.name, code);
    void load();
  }}
/>
<CodeCard bind:open={codeOpen} terminal={codeFor} {issued} />

<Modal open={renameFor !== null} title="Rename till" onclose={() => (renameFor = null)}>
  <TextField label="Name" bind:value={newName} maxlength={80} />
  {#snippet footer()}
    <Button variant="secondary" onclick={() => (renameFor = null)}>Cancel</Button>
    <Button onclick={rename} disabled={!newName.trim()}>Save</Button>
  {/snippet}
</Modal>

<style>
  .actions {
    display: inline-flex;
    gap: var(--space-1);
    justify-content: flex-end;
    flex-wrap: wrap;
  }
</style>
