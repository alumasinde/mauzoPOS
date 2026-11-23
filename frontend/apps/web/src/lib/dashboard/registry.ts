import type { WidgetEntry } from './types';
import { formatMoney } from '@mauzo/utils';
import InventoryAlerts from './widgets/InventoryAlerts.svelte';
import HeldSales from './widgets/HeldSales.svelte';
import PaymentMethods from './widgets/PaymentMethods.svelte';
import RecentSales from './widgets/RecentSales.svelte';
import SalesOverview from './widgets/SalesOverview.svelte';
import TerminalStatus from './widgets/TerminalStatus.svelte';
import { delta, fetchDaily, fetchMethods, fetchSummary } from './widgets/salesApi';

/**
 * Front-end half of the dashboard widget registry. The API decides which
 * widgets a user sees (module enabled × permission, see the module
 * catalogue); this maps each widget key to how it renders.
 *
 * To plug a module in: give its KPI entries a `load` and its panels a
 * `component` backed by the module's own endpoints. Until then the widget
 * shows its `pending` placeholder. Keys the API sends that are not listed
 * here are skipped, so the API can ship widgets ahead of the web app.
 */
export const WIDGETS: Record<string, WidgetEntry> = {
  // sales (Phase 7)
  sales_total: {
    kind: 'kpi',
    icon: 'cart',
    tone: 'blue',
    pendingNote: '',
    load: async (p) => {
      const [cur, prev, daily] = await Promise.all([
        fetchSummary(p.period, p.branchId),
        fetchSummary(p.period, p.branchId, true),
        fetchDaily(p.period, p.branchId),
      ]);
      return {
        value: formatMoney(cur.total),
        delta: delta(Number(cur.total), Number(prev.total)),
        trend: daily.map((d) => Number(d.total)),
      };
    },
  },
  sales_count: {
    kind: 'kpi',
    icon: 'receipt',
    tone: 'green',
    pendingNote: '',
    load: async (p) => {
      const [cur, prev, daily] = await Promise.all([
        fetchSummary(p.period, p.branchId),
        fetchSummary(p.period, p.branchId, true),
        fetchDaily(p.period, p.branchId),
      ]);
      return {
        value: String(cur.cnt),
        delta: delta(cur.cnt, prev.cnt),
        trend: daily.map((d) => d.cnt),
      };
    },
  },
  sales_average: {
    kind: 'kpi',
    icon: 'layers',
    tone: 'purple',
    pendingNote: '',
    load: async (p) => {
      const [cur, prev] = await Promise.all([
        fetchSummary(p.period, p.branchId),
        fetchSummary(p.period, p.branchId, true),
      ]);
      return {
        value: formatMoney(cur.avg),
        delta: delta(Number(cur.avg), Number(prev.avg)),
        trend: [],
      };
    },
  },
  sales_overview: {
    kind: 'panel',
    pending: { icon: 'chart', note: 'Sales over time appears here once sales are recorded.' },
    component: SalesOverview,
  },
  recent_sales: {
    kind: 'panel',
    pending: { icon: 'receipt', note: 'The latest transactions appear here once the POS records sales.' },
    component: RecentSales,
  },
  held_sales: {
    kind: 'panel',
    pending: { icon: 'pause', note: 'Sales parked at the till show here so they are not forgotten.' },
    component: HeldSales,
  },
  // mpesa (Phase 9)
  mpesa_collections: {
    kind: 'kpi',
    icon: 'phone',
    tone: 'teal',
    pendingNote: 'Live once M-Pesa is connected',
  },
  // payments (Phase 7)
  cash_collections: {
    kind: 'kpi',
    icon: 'wallet',
    tone: 'orange',
    pendingNote: '',
    load: async (p) => {
      const [cur, prev] = await Promise.all([
        fetchMethods(p.period, p.branchId),
        fetchMethods(p.period, p.branchId, true),
      ]);
      const cash = cur.find((m) => m.method === 'cash');
      const prevCash = prev.find((m) => m.method === 'cash');
      return {
        value: formatMoney(cash?.total ?? '0'),
        delta: delta(Number(cash?.total ?? 0), Number(prevCash?.total ?? 0)),
        trend: [],
      };
    },
  },
  payment_methods: {
    kind: 'panel',
    pending: {
      icon: 'wallet',
      note: 'The split between M-Pesa, cash and card appears here once payments are recorded.',
    },
    component: PaymentMethods,
  },
  // inventory (live)
  inventory_alerts: {
    kind: 'panel',
    pending: { icon: 'alert', note: '' },
    component: InventoryAlerts,
  },
  // terminals (live)
  terminal_status: {
    kind: 'panel',
    pending: { icon: 'monitor', note: '' },
    component: TerminalStatus,
  },
};
