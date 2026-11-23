<script lang="ts">
  import { ApiError } from '@mauzo/api-client';
  import type { Unit, UnitKind } from '@mauzo/types';
  import { Badge, Button, DataTable, Modal, TextField, toast } from '@mauzo/ui';
  import { api } from '#lib/api.ts';

  /** Units-of-measure table + the "new unit" modal (units are create-only —
   *  built-ins and conversions can't be edited in place). */
  interface Props {
    units: Unit[];
    loading: boolean;
    canManage: boolean;
    onchange: () => void;
  }
  let { units, loading, canManage, onchange }: Props = $props();

  let unitOpen = $state(false);
  let unitCode = $state('');
  let unitName = $state('');
  let unitKind = $state<UnitKind>('count');
  let unitBase = $state('');
  let unitFactor = $state('1');
  let saving = $state(false);

  const baseUnits = $derived(units.filter((u) => u.built_in && u.base_code === null && u.kind === unitKind));
  const unitColumns = [
    { key: 'code', label: 'Code', width: '100px' },
    { key: 'name', label: 'Name' },
    { key: 'kind', label: 'Kind', width: '100px' },
    { key: 'conv', label: 'Conversion', width: '160px' },
  ];

  function openNew() {
    unitCode = '';
    unitName = '';
    unitKind = 'count';
    unitBase = '';
    unitFactor = '1';
    unitOpen = true;
  }

  async function saveUnit() {
    saving = true;
    try {
      await api.post('/api/v1/t/catalog/units', {
        code: unitCode,
        name: unitName,
        kind: unitKind,
        base_code: unitBase,
        factor: unitFactor,
      });
      toast.success('Unit added');
      unitOpen = false;
      onchange();
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Save failed');
    } finally {
      saving = false;
    }
  }
</script>

<section>
  <header>
    <h2>Units of measure</h2>
    {#if canManage}<Button size="sm" onclick={openNew}>+ New</Button>{/if}
  </header>
  <DataTable columns={unitColumns} rows={units} rowKey={(r) => r.id} {loading} empty="No units.">
    {#snippet cell(row, col)}
      {#if col.key === 'conv'}
        {#if row.base_code}1 {row.code} = {row.factor} {row.base_code}{:else}base unit{/if}
      {:else if col.key === 'code'}
        {row.code}
        {#if row.built_in}<Badge tone="neutral">built-in</Badge>{/if}
      {:else}
        {(row as unknown as Record<string, unknown>)[col.key] ?? ''}
      {/if}
    {/snippet}
  </DataTable>
</section>

<Modal bind:open={unitOpen} title="New unit">
  <div class="grid2">
    <TextField label="Code (e.g. crate24)" bind:value={unitCode} />
    <TextField label="Name" bind:value={unitName} />
    <label class="fld">
      <span>Kind</span>
      <select bind:value={unitKind}>
        {#each ['count', 'mass', 'volume', 'length', 'time'] as k (k)}<option value={k}>{k}</option>{/each}
      </select>
    </label>
    <label class="fld">
      <span>Base unit</span>
      <select bind:value={unitBase}>
        <option value="">— this unit is the base —</option>
        {#each baseUnits as u (u.id)}<option value={u.code}>{u.name} ({u.code})</option>{/each}
      </select>
    </label>
    {#if unitBase}<TextField label="Factor (1 unit = factor × base)" bind:value={unitFactor} />{/if}
  </div>
  {#snippet footer()}
    <Button variant="ghost" onclick={() => (unitOpen = false)}>Cancel</Button>
    <Button onclick={saveUnit} disabled={saving || !unitCode.trim() || !unitName.trim()}>Save</Button>
  {/snippet}
</Modal>

<style>
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
  .grid2 {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 0.75rem;
  }
</style>
