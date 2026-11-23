<script lang="ts">
  import { ApiError } from '@mauzo/api-client';
  import type { PermissionGroup, Role } from '@mauzo/types';
  import { Button, Modal, TextField, toast } from '@mauzo/ui';
  import { api } from '#lib/api.ts';
  import PermissionPicker from './PermissionPicker.svelte';

  interface Props {
    open: boolean;
    role: Role | null; // null = create
    groups: PermissionGroup[];
    onsaved: () => void;
  }

  let { open = $bindable(), role, groups, onsaved }: Props = $props();

  let form = $state({
    name: '',
    description: '',
    permissions: [] as string[],
    max_discount_percent: '',
    max_refund_amount: '',
    max_cash_out_amount: '',
  });
  let saving = $state(false);
  let error = $state<string | null>(null);
  const readonly = $derived(role?.is_locked ?? false);

  $effect(() => {
    if (!open) return;
    error = null;
    form = {
      name: role?.name ?? '',
      description: role?.description ?? '',
      permissions: [...(role?.permissions ?? [])],
      max_discount_percent: role?.max_discount_percent ?? '',
      max_refund_amount: role?.max_refund_amount ?? '',
      max_cash_out_amount: role?.max_cash_out_amount ?? '',
    };
  });

  async function save() {
    saving = true;
    error = null;
    const body = {
      ...form,
      name: form.name.trim(),
      max_discount_percent: form.max_discount_percent.trim() || null,
      max_refund_amount: form.max_refund_amount.trim() || null,
      max_cash_out_amount: form.max_cash_out_amount.trim() || null,
    };
    try {
      if (role) await api.patch(`/api/v1/t/roles/${role.id}`, body);
      else await api.post('/api/v1/t/roles', body);
      toast.success(role ? 'Role updated — affected users are updated immediately' : 'Role created');
      open = false;
      onsaved();
    } catch (e) {
      error = e instanceof ApiError ? e.message : 'Save failed';
    } finally {
      saving = false;
    }
  }
</script>

<Modal bind:open title={role ? `Role — ${role.name}` : 'New role'} locked={saving} size="lg">
  <div class="grid">
    {#if readonly}
      <p class="note">The owner role always holds every permission and cannot be changed.</p>
    {:else if role?.is_system && !role.customized}
      <p class="note">
        Built-in role following its template. Saving changes stops automatic updates; use “Reset” to return.
      </p>
    {/if}
    <div class="two">
      <TextField label="Name" bind:value={form.name} disabled={readonly} />
      <TextField label="Description" bind:value={form.description} disabled={readonly} />
    </div>
    <div class="three">
      <TextField
        label="Max discount %"
        inputmode="decimal"
        bind:value={form.max_discount_percent}
        hint="Blank = unlimited"
        disabled={readonly}
      />
      <TextField
        label="Max refund (KSh)"
        inputmode="decimal"
        bind:value={form.max_refund_amount}
        hint="Blank = unlimited"
        disabled={readonly}
      />
      <TextField
        label="Max cash out (KSh)"
        inputmode="decimal"
        bind:value={form.max_cash_out_amount}
        hint="Blank = unlimited"
        disabled={readonly}
      />
    </div>
    <PermissionPicker {groups} bind:selected={form.permissions} {readonly} />
    {#if error}<p class="error">{error}</p>{/if}
  </div>
  {#snippet footer()}
    <Button variant="secondary" onclick={() => (open = false)}>{readonly ? 'Close' : 'Cancel'}</Button>
    {#if !readonly}
      <Button onclick={save} loading={saving}>{role ? 'Save' : 'Create role'}</Button>
    {/if}
  {/snippet}
</Modal>

<style>
  .grid {
    display: grid;
    gap: var(--space-4);
  }
  .two {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: var(--space-3);
  }
  .three {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
    gap: var(--space-3);
  }
  .note {
    margin: 0;
    padding: var(--space-3);
    border-radius: var(--radius-sm);
    background: var(--color-surface-muted);
    font-size: var(--text-sm);
  }
  .error {
    color: var(--color-danger);
    margin: 0;
  }
</style>
