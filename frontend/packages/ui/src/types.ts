export interface Column {
  key: string;
  label: string;
  align?: 'left' | 'right' | 'center';
  width?: string;
}

/** One x position of an AreaChart. */
export interface AreaPoint {
  /** x-axis label, e.g. "2 PM" */
  label: string;
  value: number;
}

/** One slice of a DonutChart. */
export interface DonutSlice {
  label: string;
  value: number;
  /** CSS colour; defaults to the --chart-N palette. */
  color?: string;
}
