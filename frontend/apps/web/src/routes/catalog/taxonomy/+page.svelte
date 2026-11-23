<script lang="ts">
  import { onMount } from 'svelte';
  import { ApiError } from '@mauzo/api-client';
  import type { Brand, Category, Unit } from '@mauzo/types';
  import { Button, DataTable, Modal, TextField, toast } from '@mauzo/ui';
  import { api } from '#lib/api.ts';
  import { auth } from '#lib/auth.svelte.ts';
  import UnitsPanel from '#lib/catalog/UnitsPanel.svelte';

  let categories = $state<Category[]>([]);
  let brands = $state<Brand[]>([]);
  let units = $state<Unit[]>([]);
  let loading = $state(true);

  const canManage = $derived(auth.can('catalog.category.manage'));

  // ---- category modal state -------------------------------------------------
  let catOpen = $state(false);
  let catTarget = $state<Category | null>(null);
  let catName = $state('');
  let catParent = $state('');
  let catSort = $state('0');
  let saving = $state(false);

  // ---- brand modal state ----------------------------------------------------
  let brandOpen = $state(false);
  let brandTarget = $state<Brand | null>(null);
  let brandName = $state('');

  // ---- unit modal state -----------------------------------------------------

  const catColumns = [
    { key: 'name', label: 'Category' },
    { key: 'parent', label: 'Parent', width: '160px' },
    { key: 'sort', label: 'Sort', width: '70px', align: 'right' as const },
    { key: 'actions', label: '', width: '150px', align: 'right' as const },
  ];
  const brandColumns = [
    { key: 'name', label: 'Brand' },
    { key: 'actions', label: '', width: '150px', align: 'right' as const },
  ];
  const parentName = $derived.by(() => {
    const m = new Map(categories.map((c) => [c.id, c.name]));
    return (id: string | null) => (id ? (m.get(id) ?? '—') : '—');
  });

  async function load() {
    loading = true;
    try {
      [categories, brands, units] = await Promise.all([
        api.get<Category[]>('/api/v1/t/catalog/categories'),
        api.get<Brand[]>('/api/v1/t/catalog/brands'),
        api.get<Unit[]>('/api/v1/t/catalog/units'),
      ]);
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Failed to load');
    } finally {
      loading = false;
    }
  }

  function editCat(c: Category | null) {
    catTarget = c;
    catName = c?.name ?? '';
    catParent = c?.parent_id ?? '';
    catSort = String(c?.sort ?? 0);
    catOpen = true;
  }

  async function saveCat() {
    saving = true;
    try {
      const body = { name: catName, parent_id: catParent || null, sort: Number(catSort) || 0 };
      if (catTarget) await api.patch(`/api/v1/t/catalog/categories/${catTarget.id}`, body);
      else await api.post('/api/v1/t/catalog/categories', body);
      toast.success('Category saved');
      catOpen = false;
      load();
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Save failed');
    } finally {
      saving = false;
    }
  }

  async function delCat(c: Category) {
    if (!confirm(`Delete category "${c.name}"?`)) return;
    try {
      await api.del(`/api/v1/t/catalog/categories/${c.id}`);
      toast.success('Category deleted');
      load();
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Delete failed');
    }
  }

  function editBrand(b: Brand | null) {
    brandTarget = b;
    brandName = b?.name ?? '';
    brandOpen = true;
  }

  async function saveBrand() {
    saving = true;
    try {
      if (brandTarget) await api.patch(`/api/v1/t/catalog/brands/${brandTarget.id}`, { name: brandName });
      else await api.post('/api/v1/t/catalog/brands', { name: brandName });
      toast.success('Brand saved');
      brandOpen = false;
      load();
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Save failed');
    } finally {
      saving = false;
    }
  }

  async function delBrand(b: Brand) {
    if (!confirm(`Delete brand "${b.name}"?`)) return;
    try {
      await api.del(`/api/v1/t/catalog/brands/${b.id}`);
      toast.success('Brand deleted');
      load();
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Delete failed');
    }
  }

  onMount(load);
</script>

<svelte:head><title>Categories &amp; units — Mauzo</title></svelte:head>

<h1>Categories, brands &amp; units</h1>

<div class="cols">
  <section>
    <header>
      <h2>Categories</h2>
      {#if canManage}<Button size="sm" onclick={() => editCat(null)}>+ New</Button>{/if}
    </header>
    <DataTable
      columns={catColumns}
      rows={categories}
      rowKey={(r) => r.id}
      {loading}
      empty="No categories yet."
    >
      {#snippet cell(row, col)}
        {#if col.key === 'parent'}
          {parentName(row.parent_id)}
        {:else if col.key === 'actions' && canManage}
          <div class="rowbtns">
            <Button size="sm" variant="ghost" onclick={() => editCat(row)}>Edit</Button>
            <Button size="sm" variant="ghost" onclick={() => delCat(row)}>Delete</Button>
          </div>
        {:else}
          {(row as unknown as Record<string, unknown>)[col.key] ?? ''}
        {/if}
      {/snippet}
    </DataTable>
  </section>

  <section>
    <header>
      <h2>Brands</h2>
      {#if canManage}<Button size="sm" onclick={() => editBrand(null)}>+ New</Button>{/if}
    </header>
    <DataTable columns={brandColumns} rows={brands} rowKey={(r) => r.id} {loading} empty="No brands yet.">
      {#snippet cell(row, col)}
        {#if col.key === 'actions' && canManage}
          <div class="rowbtns">
            <Button size="sm" variant="ghost" onclick={() => editBrand(row)}>Edit</Button>
            <Button size="sm" variant="ghost" onclick={() => delBrand(row)}>Delete</Button>
          </div>
        {:else}
          {(row as unknown as Record<string, unknown>)[col.key] ?? ''}
        {/if}
      {/snippet}
    </DataTable>
  </section>

  <UnitsPanel {units} {loading} {canManage} onchange={load} />
</div>

<Modal bind:open={catOpen} title={catTarget ? 'Edit category' : 'New category'}>
  <TextField label="Name" bind:value={catName} />
  <label class="fld">
    <span>Parent category</span>
    <select bind:value={catParent}>
      <option value="">— none (top level) —</option>
      {#each categories.filter((c) => c.id !== catTarget?.id) as c (c.id)}<option value={c.id}>
          {c.name}
        </option>{/each}
    </select>
  </label>
  <TextField label="Sort order" type="number" bind:value={catSort} />
  {#snippet footer()}
    <Button variant="ghost" onclick={() => (catOpen = false)}>Cancel</Button>
    <Button onclick={saveCat} disabled={saving || !catName.trim()}>Save</Button>
  {/snippet}
</Modal>

<Modal bind:open={brandOpen} title={brandTarget ? 'Edit brand' : 'New brand'}>
  <TextField label="Name" bind:value={brandName} />
  {#snippet footer()}
    <Button variant="ghost" onclick={() => (brandOpen = false)}>Cancel</Button>
    <Button onclick={saveBrand} disabled={saving || !brandName.trim()}>Save</Button>
  {/snippet}
</Modal>

<style>
  h1 {
    font-size: 1.4rem;
    margin: 0 0 1rem;
  }
  .cols {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(340px, 1fr));
    gap: 1.5rem;
    align-items: start;
  }
  section header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 0.5rem;
  }
  section h2 {
    font-size: 1rem;
    margin: 0;
  }
  .rowbtns {
    display: flex;
    gap: 0.25rem;
    justify-content: flex-end;
  }
  .fld {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    font-size: 0.85rem;
    margin-top: 0.75rem;
  }
  .fld span {
    font-weight: 500;
    color: var(--text-muted, #667);
  }
  select {
    padding: 0.45rem 0.6rem;
    border: 1px solid var(--border, #d8dce3);
    border-radius: 6px;
    font: inherit;
  }
</style>
