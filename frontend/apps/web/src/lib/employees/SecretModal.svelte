<script lang="ts">
  import { ApiError } from '@mauzo/api-client';
  import type { Employee } from '@mauzo/types';
  import { Button, Modal, TextField, toast } from '@mauzo/ui';
  import { api } from '#lib/api.ts';

  interface Props {
    open: boolean;
    employee: Employee | null;
    kind: 'password' | 'pin';
    onsaved: () => void;
  }

  let { open = $bindable(), employee, kind, onsaved }: Props = $props();
  let value = $state('');
  let saving = $state(false);
  let error = $state<string | null>(null);

  $effect(() => {
    if (open) {
      value = '';
      error = null;
    }
  });

  async function save() {
    if (!employee) return;
    saving = true;
    error = null;
    try {
      await api.post(`/api/v1/t/employees/${employee.id}/${kind}`, { value });
      toast.success(kind === 'pin' ? 'PIN updated' : 'Password updated — the employee was signed out');
      open = false;
      onsaved();
    } catch (e) {
      error = e instanceof ApiError ? e.message : 'Update failed';
    } finally {
      saving = false;
    }
  }
</script>

<Modal
  bind:open
  title={`${kind === 'pin' ? 'Set POS PIN' : 'Set password'} — ${employee?.name ?? ''}`}
  locked={saving}
>
  <form
    onsubmit={(e) => {
      e.preventDefault();
      save();
    }}
  >
    {#if kind === 'pin'}
      <TextField
        label="New PIN"
        inputmode="numeric"
        autocomplete="off"
        bind:value
        hint="4–6 digits, not repeated or sequential"
        {error}
      />
    {:else}
      <TextField
        label="New password"
        type="password"
        autocomplete="new-password"
        bind:value
        hint="At least 8 characters"
        {error}
      />
    {/if}
  </form>
  {#snippet footer()}
    <Button variant="secondary" onclick={() => (open = false)}>Cancel</Button>
    <Button onclick={save} loading={saving} disabled={!value}>Save</Button>
  {/snippet}
</Modal>
