import type { WidgetEntry } from './types';
import InventoryAlerts from './widgets/InventoryAlerts.svelte';
import TerminalStatus from './widgets/TerminalStatus.svelte';

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
  sales_total: { kind: 'kpi', icon: 'cart', tone: 'blue', pendingNote: 'Live once sales are recorded' },
  sales_count: { kind: 'kpi', icon: 'receipt', tone: 'green', pendingNote: 'Live once sales are recorded' },
  sales_average: { kind: 'kpi', icon: 'layers', tone: 'purple', pendingNote: 'Live once sales are recorded' },
  sales_overview: {
    kind: 'panel',
    pending: { icon: 'chart', note: 'Sales over time appears here once sales are recorded.' },
  },
  recent_sales: {
    kind: 'panel',
    pending: { icon: 'receipt', note: 'The latest transactions appear here once the POS records sales.' },
  },
  held_sales: {
    kind: 'panel',
    pending: { icon: 'pause', note: 'Sales parked at the till show here so they are not forgotten.' },
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
    pendingNote: 'Live once payments are recorded',
  },
  payment_methods: {
    kind: 'panel',
    pending: {
      icon: 'wallet',
      note: 'The split between M-Pesa, cash and card appears here once payments are recorded.',
    },
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
