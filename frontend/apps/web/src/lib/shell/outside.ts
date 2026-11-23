/** Svelte action: calls handler on pointerdown / Escape outside the node. */
export function onClickOutside(node: HTMLElement, handler: () => void) {
  let cb = handler;
  const down = (e: PointerEvent) => {
    if (!node.contains(e.target as Node)) cb();
  };
  const key = (e: KeyboardEvent) => {
    if (e.key === 'Escape') cb();
  };
  document.addEventListener('pointerdown', down, true);
  document.addEventListener('keydown', key);
  return {
    update(next: () => void) {
      cb = next;
    },
    destroy() {
      document.removeEventListener('pointerdown', down, true);
      document.removeEventListener('keydown', key);
    },
  };
}
