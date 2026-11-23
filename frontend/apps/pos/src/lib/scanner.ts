// HID barcode-scanner capture. Scanners type very fast and end with Enter;
// slower typing is left alone so the search box still works.

export type ScanHandler = (code: string) => void;

const SCAN_GAP_MS = 35; // chars faster than this = scanner
const SCAN_END_GAP_MS = 250; // longer pause ends a scan without Enter

export function createScanner(onScan: ScanHandler) {
  let buffer = '';
  let last = 0;
  let timer: ReturnType<typeof setTimeout> | null = null;

  function flush() {
    if (buffer.length >= 6) onScan(buffer);
    buffer = '';
  }

  function onKeydown(e: KeyboardEvent) {
    const target = e.target as HTMLElement | null;
    const typing = target && (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA');
    const now = Date.now();

    if (e.key === 'Enter') {
      if (buffer.length >= 6 && now - last < SCAN_END_GAP_MS) {
        e.preventDefault();
        const code = buffer;
        buffer = '';
        onScan(code);
        return;
      }
      buffer = '';
      return;
    }
    if (e.key.length !== 1) {
      buffer = '';
      return;
    }
    // When a field has focus the user types there; only capture when focus is
    // on the page body or the char arrives scanner-fast.
    const fast = now - last < SCAN_GAP_MS;
    if (!typing && (fast || buffer.length > 0 || /[0-9]/.test(e.key))) {
      buffer += e.key;
      last = now;
      if (timer) clearTimeout(timer);
      timer = setTimeout(flush, SCAN_END_GAP_MS);
    } else {
      buffer = '';
    }
  }

  return {
    attach() {
      window.addEventListener('keydown', onKeydown, true);
    },
    detach() {
      window.removeEventListener('keydown', onKeydown, true);
    },
  };
}
