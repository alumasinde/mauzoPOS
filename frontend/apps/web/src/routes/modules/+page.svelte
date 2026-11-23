<script lang="ts">
  import { onMount } from 'svelte';
  import { ApiError } from '@mauzo/api-client';
  import type { ModuleGroup, ModuleInfo, ModuleToggleResult } from '@mauzo/types';
  import { toast } from '@mauzo/ui';
  import { api } from '#lib/api.ts';
  import { auth } from '#lib/auth.svelte.ts';
  import { nav } from '#lib/nav.svelte.ts';
  import ConfigForm from '#lib/modules/ConfigForm.svelte';
  import ModuleRow from '#lib/modules/ModuleRow.svelte';

  const GROUPS: { key: ModuleGroup; label: string }[] = [
    { key: 'core', label: 'Core' },
    { key: 'operations', label: 'Operations' },
    { key: 'finance', label: 'Finance' },
    { key: 'industry', label: 'Industry' },
    { key: 'growth', label: 'Growth' },
  ];

  let modules = $state<ModuleInfo[]>([]);
  let businessType = $state<string | null>(null);
  let loading = $state(true);
  let busy = $state<string | null>(null);
  let configOpen = $state(false);
  let configTarget = $state<ModuleInfo | null>(null);

  const names = $derived(Object.fromEntries(modules.map((m) => [m.code, m.name])));

  async function load() {
    loading = true;
    try {
      modules = await api.get<ModuleInfo[]>('/api/v1/t/modules');
      const st = await api.get<{ business_type?: string | null }>('/api/v1/t/onboarding').catch(() => null);
      businessType = st?.business_type ?? null;
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Failed to load modules');
    } finally {
      loading = false;
    }
  }

  // Permissions and the sidebar depend on enabled modules: refresh both.
  async function applied(res: ModuleToggleResult) {
    modules = res.modules;
    await auth.refresh();
    await nav.load();
  }

  async function toggle(m: ModuleInfo, on: boolean, cascade = false) {
    busy = m.code;
    try {
      const res = await api.put<ModuleToggleResult>(`/api/v1/t/modules/${m.code}`, { enabled: on, cascade });
      await applied(res);
      const others = res.changed.filter((c) => c !== m.code).map((c) => names[c] ?? c);
      toast.success(
        `${m.name} ${on ? 'enabled' : 'disabled'}${others.length ? ` (also ${on ? 'enabled' : 'disabled'}: ${others.join(', ')})` : ''}`,
      );
    } catch (e) {
      if (e instanceof ApiError && e.code === 'module_has_dependents' && !cascade) {
        const deps = ((e.details?.dependents as string[]) ?? []).map((c) => names[c] ?? c);
        if (confirm(`${deps.join(', ')} depend on ${m.name}. Disable them too?`)) {
          busy = null;
          return toggle(m, on, true);
        }
      } else {
        toast.error(e instanceof ApiError ? e.message : 'Update failed');
      }
    } finally {
      busy = null;
    }
  }

  async function saveConfig(config: Record<string, unknown>) {
    if (!configTarget) return;
    modules = await api.put<ModuleInfo[]>(`/api/v1/t/modules/${configTarget.code}/config`, { config });
  }

  onMount(load);
</script>

<section class="head">
  <h1>Modules</h1>
  <p class="muted">
    Switch features on or off for your business. Modules your plan does not include are marked; core modules
    are always on.
  </p>
</section>

{#if loading}
  <p class="muted">Loading…</p>
{:else}
  {#each GROUPS as g (g.key)}
    {@const items = modules.filter((m) => m.group === g.key)}
    {#if items.length}
      <section class="group">
        <h2>{g.label}</h2>
        <ul>
          {#each items as m (m.code)}
            <ModuleRow
              module={m}
              {names}
              {businessType}
              busy={busy === m.code}
              ontoggle={(mod, on) => toggle(mod, on)}
              onconfigure={(mod) => {
                configTarget = mod;
                configOpen = true;
              }}
            />
          {/each}
        </ul>
      </section>
    {/if}
  {/each}
{/if}

<ConfigForm bind:open={configOpen} module={configTarget} onsave={saveConfig} />

<style>
  .head h1 {
    margin: 0 0 var(--space-1);
  }
  .muted {
    color: var(--color-text-muted);
    margin: 0 0 var(--space-5);
  }
  .group {
    margin-bottom: var(--space-5);
  }
  h2 {
    font-size: var(--text-md);
    margin: 0 0 var(--space-2);
    color: var(--color-text-muted);
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }
  ul {
    list-style: none;
    margin: 0;
    padding: 0;
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
  }
</style>
