import type { Component } from 'svelte';
import type { DashboardWidget } from '@mauzo/types';
import type { Period } from './period';

/** Props every dashboard widget component receives. */
export interface WidgetProps {
  widget: DashboardWidget;
  period: Period;
  /** Selected branch public id; null = all branches. */
  branchId: string | null;
}

export type KpiTone = 'blue' | 'green' | 'purple' | 'teal' | 'orange';

/** What a KPI loader returns. */
export interface KpiData {
  value: string;
  /** % change vs the previous period. */
  delta: number | null;
  trend: number[];
}

/** A headline figure in the top row. */
export interface KpiEntry {
  kind: 'kpi';
  icon: string;
  tone: KpiTone;
  /** Shown until `load` exists (the owning module is not built yet). */
  pendingNote: string;
  load?: (p: WidgetProps) => Promise<KpiData>;
}

/** A panel in the grid below the KPI row. */
export interface PanelEntry {
  kind: 'panel';
  /** Shown until `component` exists (the owning module is not built yet). */
  pending: { icon: string; note: string };
  component?: Component<WidgetProps>;
}

export type WidgetEntry = KpiEntry | PanelEntry;
