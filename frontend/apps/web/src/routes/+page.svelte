<script lang="ts">
  import { onMount } from 'svelte';
  import { resolve } from '$app/paths';
  import { ApiError } from '@mauzo/api-client';
  import type { DashboardWidget, OnboardingState } from '@mauzo/types';
  import { Icon, PageHeader } from '@mauzo/ui';
  import { api } from '#lib/api.ts';
  import { auth } from '#lib/auth.svelte.ts';
  import { branch } from '#lib/branch.svelte.ts';
  import KpiWidget from '#lib/dashboard/KpiWidget.svelte';
  import WidgetCard from '#lib/dashboard/WidgetCard.svelte';
  import { PERIODS, type Period } from '#lib/dashboard/period.ts';
  import { WIDGETS } from '#lib/dashboard/registry.ts';
  import type { KpiEntry, PanelEntry } from '#lib/dashboard/types.ts';

  const PERIOD_KEY = 'mauzo_dashboard_period';

  let widgets = $state<DashboardWidget[]>([]);
  let loading = $state(true);
  let error = $state<string | null>(null);
  let pendingSetup = $state(false);
  let period = $state<Period>(readPeriod());

  const firstName = $derived(auth.user?.name?.split(/\s+/)[0] ?? '');
  const where = $derived(branch.current?.name ?? (branch.allowAll ? 'all your branches' : 'your store'));
  const when = $derived(
    period === 'today' ? 'today' : `over the ${PERIODS.find((p) => p.key === period)?.label.toLowerCase()}`,
  );

  // Widgets the API allows × widgets this app knows how to draw.
  const kpis = $derived(
    widgets.flatMap((w) => {
      const e = WIDGETS[w.key];
      return e?.kind === 'kpi' ? [{ w, e }] : [];
    }) as { w: DashboardWidget; e: KpiEntry }[],
  );
  const panels = $derived(
    widgets.flatMap((w) => {
      const e = WIDGETS[w.key];
      return e?.kind === 'panel' ? [{ w, e }] : [];
    }) as { w: DashboardWidget; e: PanelEntry }[],
  );

  function readPeriod(): Period {
    try {
      const p = localStorage.getItem(PERIOD_KEY);
      if (PERIODS.some((x) => x.key === p)) return p as Period;
    } catch {
      /* default */
    }
    return 'today';
  }

  function setPeriod(p: Period) {
    period = p;
    try {
      localStorage.setItem(PERIOD_KEY, p);
    } catch {
      /* not persisted */
    }
  }

  onMount(async () => {
    try {
      widgets = (await api.get<{ widgets: DashboardWidget[] }>('/api/v1/t/dashboard')).widgets;
    } catch (e) {
      error = e instanceof ApiError ? e.message : 'Could not load the dashboard';
    } finally {
      loading = false;
    }
    if (!auth.can('settings.business.update')) return;
    try {
      const st = await api.get<OnboardingState>('/api/v1/t/onboarding');
      pendingSetup = !!st && !st.done;
    } catch {
      pendingSetup = false;
    }
  });
</script>

<PageHeader
  title="Dashboard"
  sub="Welcome back{firstName ? `, ${firstName}` : ''}! Here's what's happening at {where} {when}."
  icon="dashboard"
>
  {#snippet actions()}
    <label class="period">
      <Icon name="calendar" size={18} />
      <span class="sr">Period</span>
      <select value={period} onchange={(e) => setPeriod(e.currentTarget.value as Period)}>
        {#each PERIODS as p (p.key)}
          <option value={p.key}>{p.label}</option>
        {/each}
      </select>
      <Icon name="chevron-down" size={16} />
    </label>
  {/snippet}
</PageHeader>

{#if pendingSetup}
  <a class="banner" href={resolve('/onboarding')}>
    <Icon name="rocket" size={20} /> Finish business setup <Icon name="arrow-right" size={18} />
  </a>
{/if}

{#if loading}
  <p class="muted">Loading…</p>
{:else if error}
  <p class="err">{error}</p>
{:else if kpis.length === 0 && panels.length === 0}
  <WidgetCard title="Nothing to show yet">
    <p class="muted">
      Your dashboard fills in as modules are switched on and your role allows you to see them.
      {#if auth.can('settings.module.manage')}
        <a href={resolve('/modules')}>Manage modules</a>
      {/if}
    </p>
  </WidgetCard>
{:else}
  <!-- Re-mount widgets when the period changes so each refetches. -->
  {#key period}
    {#if kpis.length}
      <section class="kpis" aria-label="Key figures">
        {#each kpis as { w, e } (w.key)}
          <KpiWidget entry={e} widget={w} {period} branchId={branch.id} />
        {/each}
      </section>
    {/if}
    {#if panels.length}
      <section class="panels">
        {#each panels as { w, e } (w.key)}
          <div class="slot {w.size}">
            {#if e.component}
              <e.component widget={w} {period} branchId={branch.id} />
            {:else}
              <WidgetCard title={w.title} pending={e.pending} />
            {/if}
          </div>
        {/each}
      </section>
    {/if}
  {/key}
{/if}

<style>
  .period {
    position: relative;
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-2) var(--space-3);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    background: var(--color-surface);
    color: var(--color-text);
    font-weight: 600;
  }
  .period select {
    appearance: none;
    border: 0;
    background: transparent;
    color: inherit;
    font: inherit;
    padding-right: var(--space-1);
    cursor: pointer;
  }
  .period select option {
    color: var(--color-text);
    background: var(--color-surface);
  }
  .period:focus-within {
    outline: 3px solid var(--color-focus);
    outline-offset: 2px;
  }
  .period select:focus-visible {
    outline: none;
  }
  .sr {
    position: absolute;
    width: 1px;
    height: 1px;
    overflow: hidden;
    clip-path: inset(50%);
  }
  .banner {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    margin-bottom: var(--space-5);
    padding: var(--space-3) var(--space-5);
    background: color-mix(in srgb, var(--color-primary) 12%, var(--color-surface));
    color: var(--color-primary);
    border: 1px solid color-mix(in srgb, var(--color-primary) 35%, transparent);
    border-radius: var(--radius-lg);
    text-decoration: none;
    font-weight: 600;
  }
  .kpis {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(210px, 1fr));
    gap: var(--space-4);
    margin-bottom: var(--space-5);
  }
  .panels {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: var(--space-4);
  }
  .slot.wide {
    grid-column: span 2;
  }
  .muted {
    color: var(--color-text-muted);
  }
  .err {
    color: var(--color-danger);
  }

  @media (max-width: 1280px) {
    .panels {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
  }
  @media (max-width: 720px) {
    .panels {
      grid-template-columns: minmax(0, 1fr);
    }
    .slot.wide {
      grid-column: auto;
    }
  }
</style>
