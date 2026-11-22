export type ToastKind = 'success' | 'error' | 'info' | 'warning';

export interface Toast {
  id: number;
  kind: ToastKind;
  message: string;
}

let nextId = 1;

/** Global toast queue rendered by <Toaster />. */
export const toasts = $state<Toast[]>([]);

export function dismissToast(id: number): void {
  const i = toasts.findIndex((t) => t.id === id);
  if (i >= 0) toasts.splice(i, 1);
}

function push(kind: ToastKind, message: string, ttlMs = kind === 'error' ? 6000 : 3500): number {
  const id = nextId++;
  toasts.push({ id, kind, message });
  setTimeout(() => dismissToast(id), ttlMs);
  return id;
}

export const toast = {
  success: (m: string) => push('success', m),
  error: (m: string) => push('error', m),
  info: (m: string) => push('info', m),
  warning: (m: string) => push('warning', m),
};
