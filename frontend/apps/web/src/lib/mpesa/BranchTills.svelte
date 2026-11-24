<script lang="ts">
  // Which branch each M-Pesa till / store number belongs to. A payment to a
  // mapped number appears only on that branch's tills; unmapped numbers (one
  // till number for the whole business) appear at every branch.
  import { onMount } from 'svelte';
  import { ApiError } from '@mauzo/api-client';
  import type { Branch } from '@mauzo/types';
  import { Button, Icon, toast } from '@mauzo/ui';
  import { api } from '#lib/api.ts';
  import { mpesaApi, type BranchTill } from './api';

  let branches = $state<Branch[]>([]);
  let rows = $state<{ short_code: string; branch_id: string }[]>([]);
  let busy = $state(false);

  onMount(async () => {
    try {
      const [b, t] = await Promise.all([
        api.get<Branch[]>('/api/v1/t/branches/accessible'),
        mpesaApi.branchTills(),
      ]);
      branches = b;
      rows = t.map((x: BranchTill) => ({ short_code: x.short_code, branch_id: x.branch_id }));
    } catch {
      /* shown empty */
    }
  });

  async function save() {
    busy = true;
    try {
      const out = await mpesaApi.saveBranchTills(rows.filter((r) => r.short_code.trim() && r.branch_id));
      rows = out.map((x) => ({ short_code: x.short_code, branch_id: x.branch_id }));
      toast.success('Till numbers saved — tills only see their own branch’s payments');
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Could not save');
    } finally {
      busy = false;
    }
  }
</script>

<h4>Till numbers per branch</h4>
<p class="hint">
  Add each branch's Buy Goods till / store number so a payment only pops up at the branch it was paid to.
  Leave empty if the whole business uses one till number.
</p>
{#each rows as r, i (i)}
  <div class="row">
    <input bind:value={r.short_code} inputmode="numeric" placeholder="Till / store number" maxlength="20" />
    <select bind:value={r.branch_id}>
      <option value="">Branch…</option>
      {#each branches as b (b.id)}<option value={b.id}>{b.name}</option>{/each}
    </select>
    <button class="x" aria-label="Remove" onclick={() => (rows = rows.filter((_, j) => j !== i))}>
      <Icon name="x" size={14} />
    </button>
  </div>
{/each}
<div class="acts">
  <Button size="sm" variant="secondary" onclick={() => (rows = [...rows, { short_code: '', branch_id: '' }])}>
    Add till number
  </Button>
  <Button size="sm" onclick={save} loading={busy}>Save till numbers</Button>
</div>

<style>
  h4 {
    margin: var(--space-4) 0 var(--space-1);
  }
  .hint {
    margin: 0 0 var(--space-2);
    font-size: var(--text-sm);
    color: var(--color-text-muted);
  }
  .row {
    display: grid;
    grid-template-columns: 1fr 1fr auto;
    gap: var(--space-2);
    margin-bottom: var(--space-2);
  }
  input,
  select {
    padding: var(--space-2);
    font: inherit;
    color: var(--color-text);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
  }
  .x {
    display: grid;
    place-items: center;
    padding: 0 var(--space-2);
    background: none;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    color: var(--color-text-muted);
    cursor: pointer;
  }
  .acts {
    display: flex;
    gap: var(--space-2);
  }
</style>
