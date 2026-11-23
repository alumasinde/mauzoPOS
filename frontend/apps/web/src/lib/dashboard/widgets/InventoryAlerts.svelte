<script lang="ts">
  import { onMount } from 'svelte';
  import { resolve } from '$app/paths';
  import { ApiError } from '@mauzo/api-client';
  import type { ExpiringRow, ReorderRow } from '@mauzo/types';
  import { Icon } from '@mauzo/ui';
  import { api } from '#lib/api.ts';
  import { branch } from '#lib/branch.svelte.ts';
  import WidgetCard from '../WidgetCard.svelte';
  import type { WidgetProps } from '../types';

  let { widget, branchId }: WidgetProps = $props();

  interface Alert {
    key: string;
    name: string;
    detail: string;
    level: 'danger' | 'warning' | 'info';
  }

  const MAX = 5;
  const EXPIRY_DAYS = 30;
  let alerts = $state<Alert[]>([]);
  let loading = $state(true);
  let error = $state<string | null>(null);

  const DAY = 86_400_000;
  const daysUntil = (d: string) => Math.ceil((new Date(d).getTime() - Date.now()) / DAY);
  const qty = (s: string) => Number(s).toLocaleString('en-KE', { maximumFractionDigits: 3 });

  async function load() {
    try {
      const [reorders, expiring] = await Promise.all([
        // Reorder levels are per branch; with "all branches" only expiry shows.
        branchId
          ? api.get<ReorderRow[]>('/api/v1/t/inventory/reorders', { query: { branch_id: branchId } })
          : Promise.resolve([] as ReorderRow[]),
        api.get<ExpiringRow[]>('/api/v1/t/inventory/expiring', { query: { days: EXPIRY_DAYS } }),
      ]);
      const low: Alert[] = reorders
        .filter((r) => Number(r.qty) <= Number(r.min_qty))
        .sort((a, b) => Number(a.qty) / Number(a.min_qty || 1) - Number(b.qty) / Number(b.min_qty || 1))
        .map((r) => ({
          key: `low-${r.product_id}-${r.variant_id ?? ''}`,
          name: r.variant_name ? `${r.product_name} — ${r.variant_name}` : r.product_name,
          detail: Number(r.qty) <= 0 ? 'Out of stock' : `Only ${qty(r.qty)} left`,
          level: Number(r.qty) <= 0 || Number(r.qty) <= Number(r.min_qty) / 2 ? 'danger' : 'warning',
        }));
      const here = branch.current?.name;
      const exp: Alert[] = expiring
        .filter((e) => e.expiry_date && (!here || e.branch_name === here))
        .map((e) => {
          const d = daysUntil(e.expiry_date!);
          return {
            key: `exp-${e.batch_no}-${e.branch_name}`,
            name: e.product_name,
            detail:
              d < 0
                ? `Expired ${-d} day${d === -1 ? '' : 's'} ago`
                : d === 0
                  ? 'Expires today'
                  : `Expiring in ${d} day${d === 1 ? '' : 's'}`,
            level: d <= 0 ? 'danger' : 'info',
          };
        });
      alerts = [...low, ...exp];
    } catch (e) {
      error = e instanceof ApiError ? e.message : 'Could not load stock alerts';
    } finally {
      loading = false;
    }
  }

  onMount(load);

  const ICON = { danger: 'alert', warning: 'alert', info: 'info' } as const;
</script>

<WidgetCard title={widget.title} href={resolve('/inventory')} {loading} {error}>
  {#if alerts.length === 0}
    <div class="ok">
      <span class="tick"><Icon name="check" size={24} strokeWidth={2.4} /></span>
      <p>No low-stock or expiring items{branchId ? ' at this branch' : ''}.</p>
    </div>
  {:else}
    <ul>
      {#each alerts.slice(0, MAX) as a (a.key)}
        <li>
          <span class="thumb"><Icon name="box" size={22} /></span>
          <span class="lvl {a.level}"><Icon name={ICON[a.level]} size={18} strokeWidth={2.2} /></span>
          <span class="txt">
            <strong>{a.name}</strong>
            <span class={a.level}>{a.detail}</span>
          </span>
        </li>
      {/each}
    </ul>
    {#if alerts.length > MAX}
      <p class="more">+{alerts.length - MAX} more</p>
    {/if}
  {/if}
</WidgetCard>

<style>
  ul {
    list-style: none;
    margin: 0;
    padding: 0;
  }
  li {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-2) 0;
    border-bottom: 1px solid var(--color-border);
  }
  li:last-child {
    border-bottom: 0;
  }
  .thumb {
    display: grid;
    place-items: center;
    width: 42px;
    height: 42px;
    flex: none;
    border-radius: var(--radius-sm);
    background: var(--color-surface-muted);
    color: var(--color-text-muted);
  }
  .lvl {
    flex: none;
  }
  .txt {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }
  .txt strong {
    font-size: var(--text-sm);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .txt span {
    font-size: var(--text-sm);
  }
  .danger {
    color: var(--color-danger);
  }
  .warning {
    color: var(--color-warning);
  }
  .info {
    color: var(--color-primary);
  }
  .txt .info {
    color: var(--color-text-muted);
  }
  .more {
    margin: var(--space-2) 0 0;
    font-size: var(--text-sm);
    color: var(--color-text-muted);
  }
  .ok {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--space-3);
    min-height: 180px;
    text-align: center;
    color: var(--color-text-muted);
  }
  .ok p {
    margin: 0;
    font-size: var(--text-sm);
  }
  .tick {
    display: grid;
    place-items: center;
    width: 48px;
    height: 48px;
    border-radius: 50%;
    background: var(--color-success);
    color: #fff;
  }
</style>
