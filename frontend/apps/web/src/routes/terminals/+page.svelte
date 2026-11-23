<script lang="ts">
  import { onMount } from 'svelte';
  import { ApiError } from '@mauzo/api-client';
  import type { Branch, PosTerminal } from '@mauzo/types';
  import { Badge, Button, DataTable, Icon, Modal, StatusDot, TextField, toast, PageHeader } from '@mauzo/ui';
  import { formatDate } from '@mauzo/utils';
  import { api } from '#lib/api.ts';

  let rows = $state<PosTerminal[]>([]);
  let branches = $state<Branch[]>([]);
  let loading = $state(true);
  let busy = $state<string | null>(null);

  // create form
  let createOpen = $state(false);
  let form = $state({ name: '', branch_id: '' });
  let saving = $state(false);
  let formError = $state<string | null>(null);

  // issued code (shown once)
  let code = $state<{ terminal: string; value: string } | null>(null);

  const columns = [
    { key: 'name', label: 'Terminal' },
    { key: 'branch', label: 'Branch' },
    { key: 'device', label: 'Device' },
    { key: 'status', label: 'Status', width: '120px' },
    { key: 'seen', label: 'Last seen', width: '170px' },
    { key: 'actions', label: '', width: '150px', align: 'right' as const },
  ];

  async function load() {
    loading = true;
    try {
      [rows, branches] = await Promise.all([
        api.get<PosTerminal[]>('/api/v1/t/terminals'),
        api.get<Branch[]>('/api/v1/t/branches').catch(() => [] as Branch[]),
      ]);
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Failed to load terminals');
    } finally {
      loading = false;
    }
  }

  async function create() {
    saving = true;
    formError = null;
    try {
      await api.post('/api/v1/t/terminals', form);
      toast.success(`Terminal "${form.name}" registered`);
      createOpen = false;
      form = { name: '', branch_id: '' };
      await load();
    } catch (e) {
      formError = e instanceof ApiError ? e.message : 'Could not create terminal';
    } finally {
      saving = false;
    }
  }

  async function issueCode(t: PosTerminal) {
    busy = t.id;
    try {
      const res = await api.post<{ code: string }>(`/api/v1/t/terminals/${t.id}/code`);
      code = { terminal: t.name, value: res.code };
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Could not issue a code');
    } finally {
      busy = null;
    }
  }

  function seen(t: PosTerminal): string {
    if (!t.last_seen_at) return t.status === 'active' ? 'Never' : '—';
    return formatDate(t.last_seen_at);
  }

  onMount(load);
</script>

<PageHeader
  title="POS terminals"
  sub="Windows tills bound to branches. Issue a one-time code to activate a device."
  icon="monitor"
>
  {#snippet actions()}
    <Button onclick={() => (createOpen = true)}>
      <Icon name="plus" size={16} /> Register terminal
    </Button>
  {/snippet}
</PageHeader>

<DataTable {columns} {rows} rowKey={(t) => t.id} {loading} empty="No terminals registered yet.">
  {#snippet cell(t: PosTerminal, col)}
    {#if col.key === 'name'}<strong>{t.name}</strong>
    {:else if col.key === 'branch'}{t.branch ?? `#${t.branch_id}`}
    {:else if col.key === 'device'}{t.device_name ?? '—'}
    {:else if col.key === 'status'}
      <Badge tone={t.status === 'active' ? 'success' : t.status === 'pending' ? 'warning' : 'danger'}>
        {t.status}
      </Badge>
      {#if t.status === 'active'}
        <StatusDot status={t.online ? 'ok' : 'unknown'} label={t.online ? 'Online' : 'Offline'} />
      {/if}
    {:else if col.key === 'seen'}{seen(t)}
    {:else if col.key === 'actions'}
      <span class="actions">
        <Button
          size="sm"
          variant="secondary"
          disabled={busy === t.id || t.status === 'disabled'}
          onclick={() => issueCode(t)}
        >
          {busy === t.id ? 'Issuing…' : 'Issue code'}
        </Button>
      </span>
    {/if}
  {/snippet}
</DataTable>

<Modal bind:open={createOpen} title="Register terminal" locked={saving}>
  <form
    onsubmit={(e) => {
      e.preventDefault();
      create();
    }}
  >
    <TextField label="Name" bind:value={form.name} required hint="e.g. Till 1, Front counter" />
    <label class="field">
      <span>Branch</span>
      <select bind:value={form.branch_id} required>
        <option value="" disabled>Select a branch</option>
        {#each branches as b (b.id)}<option value={b.id}>{b.name}</option>{/each}
      </select>
    </label>
    {#if formError}<p class="err">{formError}</p>{/if}
    <div class="modal-actions">
      <Button variant="secondary" onclick={() => (createOpen = false)}>Cancel</Button>
      <Button type="submit" disabled={saving || !form.name || !form.branch_id}>
        {saving ? 'Registering…' : 'Register'}
      </Button>
    </div>
  </form>
</Modal>

<Modal open={code !== null} title="Activation code" onclose={() => (code = null)}>
  {#if code}
    <p>
      Type this code into <strong>{code.terminal}</strong>'s activation screen. It is single-use and will not
      be shown again.
    </p>
    <div class="code">{code.value}</div>
    <div class="modal-actions">
      <Button onclick={() => navigator.clipboard?.writeText(code?.value ?? '')}>Copy</Button>
      <Button variant="secondary" onclick={() => (code = null)}>Done</Button>
    </div>
  {/if}
</Modal>

<style>
  .actions {
    text-align: right;
  }
  .field {
    display: block;
    margin-top: 12px;
  }
  .field span {
    display: block;
    font-size: 12px;
    color: var(--ink-3, #8a93a6);
    margin-bottom: 4px;
  }
  select {
    width: 100%;
    padding: 9px 10px;
    border-radius: 8px;
    border: 1px solid var(--line, #e3e7ee);
    font: inherit;
  }
  .err {
    color: var(--danger, #d64545);
    font-size: 13px;
  }
  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    margin-top: 18px;
  }
  .code {
    font-size: 28px;
    font-weight: 700;
    letter-spacing: 2px;
    text-align: center;
    background: var(--surface-2, #f4f6fa);
    border: 1px dashed var(--line, #e3e7ee);
    border-radius: 10px;
    padding: 18px;
    margin: 14px 0;
    font-family: monospace;
  }
</style>
