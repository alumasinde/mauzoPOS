<script lang="ts">
  import './taxonomy.css';
  import { onMount } from 'svelte';
  import { ApiError } from '@mauzo/api-client';
  import type { Brand, Category, Unit } from '@mauzo/types';
  import { Button, Icon, Modal, PageHeader, Select, TextField, toast } from '@mauzo/ui';
  import { categoryIcon } from '@mauzo/utils';
  import { api } from '#lib/api.ts';
  import { auth } from '#lib/auth.svelte.ts';
  import UnitsPanel from '#lib/catalog/UnitsPanel.svelte';

  type Tab = 'categories' | 'brands' | 'units';
  let tab = $state<Tab>('categories');

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

  const TABS: { key: Tab; label: string }[] = [
    { key: 'categories', label: 'Categories' },
    { key: 'brands', label: 'Brands' },
    { key: 'units', label: 'Units of measure' },
  ];
</script>

<svelte:head><title>Catalogue settings — Mauzo</title></svelte:head>

<div class="taxset">
  <PageHeader
    title="Catalogue settings"
    sub="Manage product categories, brands and units of measure"
    icon="tags"
  />

  <div class="tabs" role="tablist">
    {#each TABS as t (t.key)}
      <button role="tab" aria-selected={tab === t.key} class:on={tab === t.key} onclick={() => (tab = t.key)}>
        {t.label}
      </button>
    {/each}
  </div>

  {#if tab === 'categories'}
    <section class="panel">
      <header class="phead">
        <h2>Categories</h2>
        {#if canManage}
          <Button size="sm" onclick={() => editCat(null)}><Icon name="plus" size={14} /> New category</Button>
        {/if}
      </header>
      {#if loading}
        <p class="muted">Loading…</p>
      {:else if categories.length === 0}
        <p class="muted">No categories yet.</p>
      {:else}
        <ul class="rows">
          {#each categories as c (c.id)}
            <li class="row">
              <span class="chip" aria-hidden="true"><Icon name={categoryIcon(c.name)} size={18} /></span>
              <span class="rname">{c.name}</span>
              <span class="rmeta">{parentName(c.parent_id)}</span>
              <span class="rcount">{c.product_count}</span>
              {#if canManage}
                <span class="rbtns">
                  <Button size="sm" variant="ghost" onclick={() => editCat(c)}>Edit</Button>
                  <Button size="sm" variant="ghost" onclick={() => delCat(c)}>Delete</Button>
                </span>
              {/if}
            </li>
          {/each}
        </ul>
      {/if}
    </section>
  {:else if tab === 'brands'}
    <section class="panel">
      <header class="phead">
        <h2>Brands</h2>
        {#if canManage}
          <Button size="sm" onclick={() => editBrand(null)}><Icon name="plus" size={14} /> New brand</Button>
        {/if}
      </header>
      {#if loading}
        <p class="muted">Loading…</p>
      {:else if brands.length === 0}
        <p class="muted">No brands yet.</p>
      {:else}
        <ul class="rows">
          {#each brands as b (b.id)}
            <li class="row">
              <span class="chip brand" aria-hidden="true"><Icon name="star" size={17} /></span>
              <span class="rname">{b.name}</span>
              <span class="rmeta"></span>
              <span class="rcount">{b.product_count}</span>
              {#if canManage}
                <span class="rbtns">
                  <Button size="sm" variant="ghost" onclick={() => editBrand(b)}>Edit</Button>
                  <Button size="sm" variant="ghost" onclick={() => delBrand(b)}>Delete</Button>
                </span>
              {/if}
            </li>
          {/each}
        </ul>
      {/if}
    </section>
  {:else}
    <section class="panel bare">
      <UnitsPanel {units} {loading} {canManage} onchange={load} />
    </section>
  {/if}

  <Modal bind:open={catOpen} title={catTarget ? 'Edit category' : 'New category'}>
    <TextField label="Name" bind:value={catName} />
    <label class="fld">
      <span>Parent category</span>
      <Select bind:value={catParent}>
        <option value="">— none (top level) —</option>
        {#each categories.filter((c) => c.id !== catTarget?.id) as c (c.id)}<option value={c.id}>
            {c.name}
          </option>{/each}
      </Select>
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
</div>
