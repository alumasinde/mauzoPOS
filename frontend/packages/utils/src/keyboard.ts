// Keyboard shortcut helpers for the POS (Esc, F5–F10, Ctrl+K ...).

export interface Shortcut {
  /** KeyboardEvent.key, case-insensitive, e.g. "F5", "Escape", "k" */
  key: string;
  ctrl?: boolean;
  shift?: boolean;
  alt?: boolean;
  handler: (e: KeyboardEvent) => void;
}

export function matches(e: KeyboardEvent, s: Shortcut): boolean {
  return (
    e.key.toLowerCase() === s.key.toLowerCase() &&
    e.ctrlKey === !!s.ctrl &&
    e.shiftKey === !!s.shift &&
    e.altKey === !!s.alt
  );
}

/** Registers shortcuts on target; returns an unsubscribe function. */
export function bindShortcuts(shortcuts: Shortcut[], target: Window | HTMLElement = window): () => void {
  const listener = (ev: Event) => {
    const e = ev as KeyboardEvent;
    const hit = shortcuts.find((s) => matches(e, s));
    if (hit) {
      e.preventDefault();
      hit.handler(e);
    }
  };
  target.addEventListener('keydown', listener);
  return () => target.removeEventListener('keydown', listener);
}
