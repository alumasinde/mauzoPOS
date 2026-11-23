<script lang="ts">
  import { onMount } from 'svelte';
  import { SvelteURLSearchParams } from 'svelte/reactivity';
  import { ApiError } from '@mauzo/api-client';
  import type { Customer } from '@mauzo/types';
  import { Badge, Button, DataTable, Icon, Modal, TextField, toast, PageHeader } from '@mauzo/ui';
  import { formatMoney } from '@mauzo/utils';
  import { api } from '#lib/api.ts';
  import { auth } from '#lib/auth.svelte.ts';

  let rows = $state<Customer[]>([]);
  let loading = $state(true);
  let search = $state('');
  let showAll = $state(false);

  let formOpen = $state(false);
  let target = $state<Customer | null>(null);
  let form = $state({ code: '', name: '', phone: '', email: '', kra_pin: '', credit_limit: '', notes: '' });
  let saving = $state(false);
  let formError = $state<string | null>(null);

  const canManage = $derived(auth.can('customers.customer.manage'));

  const columns = [
    { key: 'name', label: 'Customer' },
    { key: 'phone', label: 'Phone', width: '140px' },
    { key: 'email', label: 'Email' },
    { key: 'credit', label: 'Credit', width: '160px', align: 'right' as const },
    { key: 'status', label: 'Status', width: '100px' },
    { key: 'actions', label: '', width: '160px', align: 'right' as const },
  ];

  async function load() {
    loading = true;
    try {
      const p = new SvelteURLSearchParams();
      if (search) p.set('search', search);
      if (showAll) p.set('all', '1');
      rows = await api.get<Customer[]>(`/api/v1/t/customers?${p}`);
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Failed to load customers');
    } finally {
      loading = false;
    }
  }

  function edit(c: Customer | null) {
    target = c;
    form = c
      ? {
          code: c.code,
          name: c.name,
          phone: c.phone,
          email: c.email,
          kra_pin: c.kra_pin,
          credit_limit: c.credit_limit ?? '',
          notes: c.notes ?? '',
        }
      : { code: '', name: '', phone: '', email: '', kra_pin: '', credit_limit: '', notes: '' };
    formError = null;
    formOpen = true;
  }

  async function save() {
    saving = true;
    formError = null;
    const body = { ...form, credit_limit: form.credit_limit || null, notes: form.notes || null };
    try {
      if (target) {
        await api.patch(`/api/v1/t/customers/${target.id}`, body);
        toast.success('Customer updated');
      } else {
        await api.post('/api/v1/t/customers', body);
        toast.success(`Customer "${form.name}" created`);
      }
      formOpen = false;
      await load();
    } catch (e) {
      formError = e instanceof ApiError ? e.message : 'Save failed';
    } finally {
      saving = false;
    }
  }

  async function archive(c: Customer, archived: boolean) {
    if (archived && !confirm(`Archive ${c.name}? Their credit sales history is kept.`)) return;
    try {
      await api.patch(`/api/v1/t/customers/${c.id}`, { archive: archived });
      toast.success(archived ? 'Customer archived' : 'Customer restored');
      await load();
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Update failed');
    }
  }

  onMount(load);
</script>

<PageHeader title="Customers" sub="Accounts for credit sales, loyalty and receipts." icon="users">
  {#snippet actions()}
    {#if canManage}
      <Button onclick={() => edit(null)}><Icon name="plus" size={16} /> New customer</Button>
    {/if}
  {/snippet}
</PageHeader>

<div class="filters">
  <input
    class="search"
    placeholder="Search name, phone or code…"
    bind:value={search}
    onkeydown={(e) => e.key === 'Enter' && load()}
  />
  <label class="chk"><input type="checkbox" bind:checked={showAll} onchange={load} /> show archived</label>
  <Button variant="secondary" onclick={load}>Search</Button>
</div>

<DataTable {columns} {rows} rowKey={(c) => c.id} {loading} empty="No customers yet.">
  {#snippet cell(c: Customer, col)}
    {#if col.key === 'name'}<strong>{c.name}</strong>{#if c.code}
        <span class="dim">· {c.code}</span>{/if}
    {:else if col.key === 'phone'}{c.phone || '—'}
    {:else if col.key === 'email'}{c.email || '—'}
    {:else if col.key === 'credit'}<span class="num">
        {#if Number(c.credit_balance)}<span class="owed">{formatMoney(c.credit_balance)}</span>{:else}—{/if}
        {#if c.credit_limit}<span class="dim"> / {formatMoney(c.credit_limit)}</span>{/if}
      </span>
    {:else if col.key === 'status'}<Badge tone={c.status === 'active' ? 'success' : 'neutral'}>
        {c.status}
      </Badge>
    {:else if col.key === 'actions'}
      {#if canManage}
        <span class="actions">
          <Button size="sm" variant="secondary" onclick={() => edit(c)}>Edit</Button>
          <Button size="sm" variant="secondary" onclick={() => archive(c, c.status === 'active')}>
            {c.status === 'active' ? 'Archive' : 'Restore'}
          </Button>
        </span>
      {/if}
    {/if}
  {/snippet}
</DataTable>

<Modal bind:open={formOpen} title={target ? 'Edit customer' : 'New customer'} locked={saving} size="lg">
  <form
    onsubmit={(e) => {
      e.preventDefault();
      save();
    }}
  >
    <div class="grid">
      <TextField label="Name" bind:value={form.name} required />
      <TextField label="Code" bind:value={form.code} hint="Optional short code" />
      <TextField label="Phone" bind:value={form.phone} />
      <TextField label="Email" type="email" bind:value={form.email} />
      <TextField label="KRA PIN" bind:value={form.kra_pin} />
      <TextField label="Credit limit (KSh)" bind:value={form.credit_limit} hint="Blank = no credit" />
    </div>
    <label class="field">
      <span>Notes</span>
      <textarea bind:value={form.notes} rows="2"></textarea>
    </label>
    {#if formError}<p class="err">{formError}</p>{/if}
    <div class="modal-actions">
      <Button variant="secondary" onclick={() => (formOpen = false)}>Cancel</Button>
      <Button type="submit" disabled={saving || !form.name}>{saving ? 'Saving…' : 'Save'}</Button>
    </div>
  </form>
</Modal>

<style>
  .filters {
    display: flex;
    gap: 10px;
    align-items: center;
    margin-bottom: 14px;
  }
  .search {
    flex: 1;
    min-width: 220px;
    padding: 8px 10px;
    border: 1px solid var(--line, #e3e7ee);
    border-radius: 8px;
    font: inherit;
    font-size: 13px;
  }
  .chk {
    font-size: 13px;
    color: var(--ink-3, #8a93a6);
    display: flex;
    gap: 6px;
    align-items: center;
  }
  .num {
    text-align: right;
    font-variant-numeric: tabular-nums;
  }
  .owed {
    color: var(--danger, #d64545);
    font-weight: 600;
  }
  .dim {
    color: var(--ink-3, #8a93a6);
    font-size: 12px;
  }
  .actions {
    text-align: right;
    display: flex;
    gap: 6px;
    justify-content: flex-end;
  }
  .grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 0 16px;
  }
  .field {
    display: block;
    margin-top: 8px;
  }
  .field span {
    display: block;
    font-size: 12px;
    color: var(--ink-3, #8a93a6);
    margin-bottom: 4px;
  }
  textarea {
    width: 100%;
    padding: 9px 10px;
    border-radius: 8px;
    border: 1px solid var(--line, #e3e7ee);
    font: inherit;
    resize: vertical;
  }
  .err {
    color: var(--danger, #d64545);
    font-size: 13px;
  }
  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    margin-top: 16px;
  }
</style>
