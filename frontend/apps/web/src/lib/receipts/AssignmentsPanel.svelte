<script lang="ts">
  // Which layout each branch prints. "All branches" applies wherever a branch
  // has no layout of its own; with neither, the business-type default prints.
  import { onMount } from 'svelte';
  import { ApiError } from '@mauzo/api-client';
  import type { Branch, ReceiptAssignment, ReceiptTemplate } from '@mauzo/types';
  import { toast } from '@mauzo/ui';
  import { api } from '#lib/api.ts';
  import { auth } from '#lib/auth.svelte.ts';

  let { templates }: { templates: ReceiptTemplate[] } = $props();

  let branches = $state<Branch[]>([]);
  let rows = $state<ReceiptAssignment[]>([]);
  let busy = $state<string | null>(null);

  const byBranch = $derived(Object.fromEntries(rows.map((r) => [r.branch_id, r.template_id])));
  const allBranches = $derived(auth.user?.all_branches ?? false);

  onMount(async () => {
    try {
      [branches, rows] = await Promise.all([
        api.get<Branch[]>('/api/v1/t/branches/accessible'),
        api.get<ReceiptAssignment[]>('/api/v1/t/receipts/assignments'),
      ]);
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Could not load branches');
    }
  });

  async function assign(branchId: string, templateId: string) {
    busy = branchId || 'all';
    try {
      rows = await api.put<ReceiptAssignment[]>('/api/v1/t/receipts/assignments', {
        branch_id: branchId,
        template_id: templateId,
      });
      toast.success('Saved — tills pick it up on their next sync');
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Could not save');
    } finally {
      busy = null;
    }
  }
</script>

<div class="panel">
  <p class="muted">
    Choose the layout each branch prints. Tills download it with their next sync and keep printing it offline.
  </p>
  <table>
    <thead><tr><th>Branch</th><th>Layout</th></tr></thead>
    <tbody>
      {#if allBranches}
        <tr>
          <td>
            <strong>All branches</strong><br /><small>Used where a branch has no layout of its own</small>
          </td>
          <td>
            <select
              value={byBranch[''] ?? ''}
              disabled={busy === 'all'}
              onchange={(e) => assign('', e.currentTarget.value)}
            >
              <option value="">Business-type default</option>
              {#each templates as t (t.id)}<option value={t.id}>{t.name}</option>{/each}
            </select>
          </td>
        </tr>
      {/if}
      {#each branches as b (b.id)}
        <tr>
          <td>
            {b.name}{#if b.is_main}<small> · main</small>{/if}
          </td>
          <td>
            <select
              value={byBranch[b.id] ?? ''}
              disabled={busy === b.id}
              onchange={(e) => assign(b.id, e.currentTarget.value)}
            >
              <option value="">{allBranches ? 'Same as all branches' : 'Default'}</option>
              {#each templates as t (t.id)}<option value={t.id}>{t.name}</option>{/each}
            </select>
          </td>
        </tr>
      {/each}
    </tbody>
  </table>
</div>

<style>
  .muted {
    color: var(--color-text-muted);
    margin: 0 0 var(--space-4);
  }
  table {
    width: 100%;
    border-collapse: collapse;
  }
  th {
    text-align: left;
    font-size: var(--text-xs);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--color-text-muted);
    padding: var(--space-2);
    border-bottom: 1px solid var(--color-border);
  }
  td {
    padding: var(--space-3) var(--space-2);
    border-bottom: 1px solid var(--color-border);
    vertical-align: middle;
  }
  td small {
    color: var(--color-text-muted);
  }
  select {
    width: 100%;
    max-width: 320px;
    padding: var(--space-2);
    font: inherit;
    color: var(--color-text);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
  }
</style>
