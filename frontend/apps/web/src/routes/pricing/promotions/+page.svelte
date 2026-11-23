<script lang="ts">
  import { onMount } from 'svelte';
  import { ApiError } from '@mauzo/api-client';
  import type { Promotion } from '@mauzo/types';
  import { Badge, Button, DataTable, toast, PageHeader } from '@mauzo/ui';
  import { api } from '#lib/api.ts';
  import { auth } from '#lib/auth.svelte.ts';
  import PromoForm from '#lib/pricing/PromoForm.svelte';

  const columns = [
    { key: 'name', label: 'Name' },
    { key: 'rule', label: 'Rule', width: '170px' },
    { key: 'window', label: 'Window', width: '190px' },
    { key: 'targets', label: 'Targets', width: '110px' },
    { key: 'is_active', label: 'Status', width: '90px' },
    { key: 'actions', label: '', width: '160px', align: 'right' as const },
  ];

  let rows = $state<Promotion[]>([]);
  let loading = $state(true);
  let open = $state(false);
  let target = $state<Promotion | null>(null);
  let busy = $state<string | null>(null);
  const canManage = $derived(auth.can('promotions.promotion.manage'));

  async function load() {
    loading = true;
    try {
      rows = await api.get<Promotion[]>('/api/v1/t/promotions/');
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Failed to load promotions');
    } finally {
      loading = false;
    }
  }

  async function remove(row: Promotion) {
    busy = row.id;
    try {
      await api.del(`/api/v1/t/promotions/${row.id}`);
      toast.success('Deleted');
      load();
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Delete failed');
    } finally {
      busy = null;
    }
  }

  function rule(p: Promotion) {
    if (p.kind === 'percent') return `${p.config.percent}% off`;
    if (p.kind === 'amount') return `KES ${p.config.amount} off/unit`;
    return `Buy ${p.config.buy} get ${p.config.get} at ${p.config.get_percent}% off`;
  }

  function window_(p: Promotion) {
    const f = (s?: string | null) => (s ? new Date(s).toLocaleDateString() : '');
    if (!p.starts_at && !p.ends_at) return 'always';
    return `${f(p.starts_at) || '…'} → ${f(p.ends_at) || '…'}`;
  }

  onMount(load);
</script>

<PageHeader
  title="Promotions"
  sub="Percent, amount and buy-X-get-Y discounts with windows and targets."
  icon="percent"
>
  {#snippet actions()}
    {#if canManage}
      <Button
        onclick={() => {
          target = null;
          open = true;
        }}
      >
        New promotion
      </Button>
    {/if}
  {/snippet}
</PageHeader>

<DataTable {columns} {rows} rowKey={(r) => r.id} {loading} empty="No promotions yet.">
  {#snippet cell(row, col)}
    {#if col.key === 'rule'}
      {rule(row)}
    {:else if col.key === 'window'}
      {window_(row)}
    {:else if col.key === 'targets'}
      {row.targets.length === 0 ? 'All products' : row.targets.length}
    {:else if col.key === 'is_active'}
      <Badge tone={row.is_active ? 'success' : 'neutral'}>{row.is_active ? 'active' : 'off'}</Badge>
    {:else if col.key === 'actions' && canManage}
      <Button
        size="sm"
        variant="ghost"
        onclick={() => {
          target = row;
          open = true;
        }}
      >
        Edit
      </Button>
      <Button size="sm" variant="ghost" disabled={busy !== null} onclick={() => remove(row)}>Delete</Button>
    {:else}
      {(row as unknown as Record<string, unknown>)[col.key] ?? '—'}
    {/if}
  {/snippet}
</DataTable>

<PromoForm {target} bind:open ondone={load} onclose={() => (open = false)} />

<style>
</style>
