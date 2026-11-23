<script lang="ts">
  import { ApiError } from '@mauzo/api-client';
  import type { Branch, Employee, Ref } from '@mauzo/types';
  import { Button, Modal, TextField, toast } from '@mauzo/ui';
  import { api } from '#lib/api.ts';
  import { auth } from '#lib/auth.svelte.ts';

  interface Props {
    open: boolean;
    employee: Employee | null; // null = create
    roles: Ref[]; // assignable roles
    branches: Branch[];
    onsaved: () => void;
  }

  let { open = $bindable(), employee, roles, branches, onsaved }: Props = $props();

  let form = $state({
    name: '',
    email: '',
    username: '',
    password: '',
    pin: '',
    role_id: '',
    all_branches: true,
    branch_ids: [] as string[],
  });
  let saving = $state(false);
  let error = $state<string | null>(null);

  const isSelf = $derived(employee !== null && employee.id === auth.user?.id);
  const title = $derived(employee ? `Edit — ${employee.name}` : 'New employee');

  // Reset the form whenever the modal opens for a different target.
  $effect(() => {
    if (!open) return;
    error = null;
    form = {
      name: employee?.name ?? '',
      email: employee?.email ?? '',
      username: employee?.username ?? '',
      password: '',
      pin: '',
      role_id: employee?.role.id ?? roles.find((r) => r.code === 'cashier')?.id ?? '',
      all_branches: employee?.all_branches ?? false,
      branch_ids: employee?.branches.map((b) => b.id) ?? (branches.length === 1 ? [branches[0]!.id] : []),
    };
  });

  function toggleBranch(id: string, on: boolean) {
    form.branch_ids = on ? [...form.branch_ids, id] : form.branch_ids.filter((b) => b !== id);
  }

  async function save() {
    saving = true;
    error = null;
    const body: Record<string, unknown> = {
      name: form.name.trim(),
      email: form.email.trim(),
      username: form.username.trim(),
    };
    if (!isSelf) {
      body.role_id = form.role_id;
      body.all_branches = form.all_branches;
      if (!form.all_branches) body.branch_ids = form.branch_ids;
    }
    try {
      if (employee) {
        await api.patch(`/api/v1/t/employees/${employee.id}`, body);
      } else {
        if (form.password) body.password = form.password;
        if (form.pin) body.pin = form.pin;
        await api.post('/api/v1/t/employees', body);
      }
      toast.success(employee ? 'Employee updated' : 'Employee added');
      open = false;
      onsaved();
    } catch (e) {
      error = e instanceof ApiError ? e.message : 'Save failed';
    } finally {
      saving = false;
    }
  }
</script>

<Modal bind:open {title} locked={saving} size="lg">
  <form
    class="grid"
    onsubmit={(e) => {
      e.preventDefault();
      save();
    }}
  >
    <TextField label="Full name" bind:value={form.name} required />
    <div class="two">
      <TextField label="Email" type="email" bind:value={form.email} hint="For back-office sign-in" />
      <TextField label="Username" bind:value={form.username} hint="For staff without email" />
    </div>
    {#if !employee}
      <div class="two">
        <TextField
          label="Password"
          type="password"
          bind:value={form.password}
          hint="Optional — at least 8 characters"
          autocomplete="new-password"
        />
        <TextField
          label="POS PIN"
          inputmode="numeric"
          bind:value={form.pin}
          hint="Optional — 4–6 digits"
          autocomplete="off"
        />
      </div>
    {/if}

    <label class="field">
      <span>Role</span>
      <select bind:value={form.role_id} disabled={isSelf}>
        {#if employee && !roles.some((r) => r.id === employee.role.id)}
          <option value={employee.role.id}>{employee.role.name}</option>
        {/if}
        {#each roles as r (r.id)}
          <option value={r.id}>{r.name}</option>
        {/each}
      </select>
    </label>

    <fieldset disabled={isSelf}>
      <legend>Branch access</legend>
      <label class="check">
        <input type="checkbox" bind:checked={form.all_branches} /> All branches
      </label>
      {#if !form.all_branches}
        {#each branches as b (b.id)}
          <label class="check">
            <input
              type="checkbox"
              checked={form.branch_ids.includes(b.id)}
              onchange={(e) => toggleBranch(b.id, e.currentTarget.checked)}
            />
            {b.name} <span class="muted">{b.code}</span>
          </label>
        {:else}
          <p class="muted">No branches yet — add one in Setup.</p>
        {/each}
      {/if}
    </fieldset>
    {#if isSelf}
      <p class="muted">You cannot change your own role or branch access.</p>
    {/if}
    {#if error}<p class="error">{error}</p>{/if}
    <button type="submit" hidden aria-hidden="true"></button>
  </form>

  {#snippet footer()}
    <Button variant="secondary" onclick={() => (open = false)}>Cancel</Button>
    <Button onclick={save} loading={saving}>{employee ? 'Save' : 'Add employee'}</Button>
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
  .field {
    display: grid;
    gap: var(--space-1);
    font-size: var(--text-sm);
    font-weight: 600;
  }
  select {
    min-height: 40px;
    padding: var(--space-2) var(--space-3);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    background: var(--color-surface);
    color: var(--color-text);
    font: inherit;
  }
  fieldset {
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    padding: var(--space-3) var(--space-4);
    display: grid;
    gap: var(--space-2);
  }
  legend {
    font-size: var(--text-sm);
    font-weight: 600;
    padding: 0 var(--space-1);
  }
  .check {
    display: flex;
    gap: var(--space-2);
    align-items: center;
  }
  .muted {
    color: var(--color-text-muted);
    font-size: var(--text-sm);
    margin: 0;
  }
  .error {
    color: var(--color-danger);
    margin: 0;
  }
</style>
