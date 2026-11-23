<script lang="ts">
  import { ApiError } from '@mauzo/api-client';
  import type { ModuleInfo, ModuleToggleResult, Tenant } from '@mauzo/types';
  import { Badge, Button, Modal, Switch, toast } from '@mauzo/ui';
  import { api } from '#lib/api.ts';
  import { auth } from '#lib/auth.svelte.ts';

  let { open = $bindable(), tenant }: { open: boolean; tenant: Tenant | null } = $props();

  let rows = $state<ModuleInfo[]>([]);
  let loading = $state(false);
  let busy = $state<string | null>(null);
  const canManage = $derived(auth.can('platform.module.manage'));

  $effect(() => {
    if (open && tenant) load(tenant.id);
  });

  async function load(id: string) {
    loading = true;
    try {
      rows = await api.get<ModuleInfo[]>(`/api/v1/platform/tenants/${id}/modules`);
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Failed to load modules');
    } finally {
      loading = false;
    }
  }

  // Platform overrides bypass the tenant's plan; disabling cascades.
  async function toggle(m: ModuleInfo, enabled: boolean) {
    if (!tenant) return;
    busy = m.code;
    try {
      const res = await api.put<ModuleToggleResult>(
        `/api/v1/platform/tenants/${tenant.id}/modules/${m.code}`,
        {
          enabled,
        },
      );
      rows = res.modules;
      toast.success(`${m.name} ${enabled ? 'enabled' : 'disabled'} for ${tenant.name}`);
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Update failed');
    } finally {
      busy = null;
    }
  }
</script>

<Modal bind:open title={`Modules — ${tenant?.name ?? ''}`}>
  {#if loading}
    <p class="muted">Loading…</p>
  {:else}
    <p class="muted">Overrides set here bypass the tenant's plan. Core modules are always on.</p>
    <ul>
      {#each rows as m (m.code)}
        <li>
          <span class="name">
            {m.name}
            {#if m.source === 'platform'}<Badge>override</Badge>{/if}
            {#if m.enabled && !m.effective}<Badge tone="warning">{m.reason?.replace(/_/g, ' ')}</Badge>{/if}
            {#if m.in_plan === false}<Badge tone="warning">not in plan</Badge>{/if}
          </span>
          <Switch
            checked={!!m.enabled}
            label={`Toggle ${m.name}`}
            disabled={m.core || !canManage}
            busy={busy === m.code}
            onchange={(on) => toggle(m, on)}
          />
        </li>
      {/each}
    </ul>
  {/if}
  {#snippet footer()}
    <Button variant="secondary" onclick={() => (open = false)}>Close</Button>
  {/snippet}
</Modal>

<style>
  .muted {
    color: var(--color-text-muted);
    margin: 0 0 var(--space-3);
    font-size: var(--text-sm);
  }
  ul {
    list-style: none;
    margin: 0;
    padding: 0;
    max-height: 60vh;
    overflow: auto;
  }
  li {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: var(--space-2) 0;
    border-bottom: 1px solid var(--color-border);
  }
  .name {
    display: flex;
    gap: var(--space-2);
    align-items: center;
  }
</style>
