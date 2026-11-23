<script lang="ts">
  import { onMount } from 'svelte';
  import { ApiError } from '@mauzo/api-client';
  import type { Branch, PriceList } from '@mauzo/types';
  import { Badge, Button, DataTable, toast, PageHeader } from '@mauzo/ui';
  import { api } from '#lib/api.ts';
  import { auth } from '#lib/auth.svelte.ts';
  import ListForm from '#lib/pricing/ListForm.svelte';

  const columns = [
    { key: 'name', label: 'Name' },
    { key: 'scope', label: 'Scope', width: '200px' },
    { key: 'priority', label: 'Priority', width: '80px', align: 'right' as const },
    { key: 'window', label: 'Window', width: '190px' },
    { key: 'is_active', label: 'Status', width: '90px' },
    { key: 'actions', label: '', width: '160px', align: 'right' as const },
  ];

  let rows = $state<PriceList[]>([]);
  let branches = $state<Branch[]>([]);
  let loading = $state(true);
  let open = $state(false);
  let target = $state<PriceList | null>(null);
  let busy = $state<string | null>(null);
  const canEdit = $derived(auth.can('pricing.price.update'));

  async function load() {
    loading = true;
    try {
      rows = await api.get<PriceList[]>('/api/v1/t/pricing/lists');
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Failed to load price lists');
    } finally {
      loading = false;
    }
  }

  async function edit(row: PriceList) {
    try {
      target = await api.get<PriceList>(`/api/v1/t/pricing/lists/${row.id}`);
      open = true;
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Failed to load list');
    }
  }

  async function remove(row: PriceList) {
    busy = row.id;
    try {
      await api.del(`/api/v1/t/pricing/lists/${row.id}`);
      toast.success('Deleted');
      load();
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Delete failed');
    } finally {
      busy = null;
    }
  }

  function scope(row: PriceList) {
    const b = branches.find((x) => x.id === row.branch_id)?.name;
    const g = row.customer_group;
    return [b ?? 'All branches', g ?? 'everyone'].join(' · ');
  }

  function window_(row: PriceList) {
    const f = (s?: string | null) => (s ? new Date(s).toLocaleDateString() : '');
    if (!row.valid_from && !row.valid_to) return 'always';
    return `${f(row.valid_from) || '…'} → ${f(row.valid_to) || '…'}`;
  }

  onMount(async () => {
    try {
      branches = await api.get<Branch[]>('/api/v1/t/branches');
    } catch {
      /* optional */
    }
    load();
  });
</script>

<PageHeader title="Price lists" sub="Branch and customer-group pricing with quantity tiers." icon="tag">
  {#snippet actions()}
    {#if canEdit}
      <Button
        onclick={() => {
          target = null;
          open = true;
        }}
      >
        New price list
      </Button>
    {/if}
  {/snippet}
</PageHeader>

<DataTable
  {columns}
  {rows}
  rowKey={(r) => r.id}
  {loading}
  empty="No price lists — the catalog base price is always used."
>
  {#snippet cell(row, col)}
    {#if col.key === 'scope'}
      {scope(row)}
    {:else if col.key === 'window'}
      {window_(row)}
    {:else if col.key === 'is_active'}
      <Badge tone={row.is_active ? 'success' : 'neutral'}>{row.is_active ? 'active' : 'off'}</Badge>
    {:else if col.key === 'actions' && canEdit}
      <Button size="sm" variant="ghost" onclick={() => edit(row)}>Edit</Button>
      <Button size="sm" variant="ghost" disabled={busy !== null} onclick={() => remove(row)}>Delete</Button>
    {:else}
      {(row as unknown as Record<string, unknown>)[col.key] ?? '—'}
    {/if}
  {/snippet}
</DataTable>

<ListForm {branches} {target} bind:open ondone={load} onclose={() => (open = false)} />

<style>
</style>
