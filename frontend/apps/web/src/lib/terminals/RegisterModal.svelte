<script lang="ts">
  // Register a till and get its activation code in one step.
  import { ApiError } from '@mauzo/api-client';
  import type { Branch, IssuedTerminalCode, PosTerminal } from '@mauzo/types';
  import { Button, Modal, TextField } from '@mauzo/ui';
  import { api } from '#lib/api.ts';
  import { branch } from '#lib/branch.svelte.ts';

  interface Props {
    open: boolean;
    branches: Branch[];
    existing: number;
    oncreated: (t: PosTerminal, code: IssuedTerminalCode) => void;
  }
  let { open = $bindable(), branches, existing, oncreated }: Props = $props();

  let name = $state('');
  let branchId = $state('');
  let saving = $state(false);
  let error = $state<string | null>(null);

  $effect(() => {
    if (!open) return;
    name = `Till ${existing + 1}`;
    branchId = branch.id ?? branches[0]?.id ?? '';
    error = null;
  });

  async function create() {
    saving = true;
    error = null;
    try {
      const t = await api.post<PosTerminal>('/api/v1/t/terminals', {
        name: name.trim(),
        branch_id: branchId,
      });
      const code = await api.post<IssuedTerminalCode>(`/api/v1/t/terminals/${t.id}/code`);
      open = false;
      oncreated(t, code);
    } catch (e) {
      error = e instanceof ApiError ? e.message : 'Could not register the till';
    } finally {
      saving = false;
    }
  }
</script>

<Modal bind:open title="Add a till" locked={saving}>
  <form
    onsubmit={(e) => {
      e.preventDefault();
      void create();
    }}
  >
    <TextField label="Name" bind:value={name} required maxlength={80} hint="Shown on receipts and reports" />
    <label class="field">
      <span>Branch</span>
      <select bind:value={branchId} required>
        {#each branches as b (b.id)}<option value={b.id}>{b.name}</option>{/each}
      </select>
    </label>
    <p class="hint">You get the activation code straight away — type it on the till to pair it.</p>
    {#if error}<p class="err">{error}</p>{/if}
  </form>
  {#snippet footer()}
    <Button variant="secondary" onclick={() => (open = false)}>Cancel</Button>
    <Button onclick={create} loading={saving} disabled={!name.trim() || !branchId}>
      Add till &amp; get code
    </Button>
  {/snippet}
</Modal>

<style>
  .field {
    display: grid;
    gap: 4px;
    margin-top: var(--space-3);
  }
  .field span {
    font-size: var(--text-sm);
    font-weight: 600;
  }
  select {
    padding: var(--space-2);
    font: inherit;
    color: var(--color-text);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
  }
  .hint {
    color: var(--color-text-muted);
    font-size: var(--text-sm);
  }
  .err {
    color: var(--color-danger);
  }
</style>
