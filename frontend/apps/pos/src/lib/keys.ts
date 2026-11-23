// Checkout keyboard shortcuts. F-keys are intercepted before the WebView can
// react to them (F5 would reload the page otherwise).
export function posKeydown(e: KeyboardEvent, act: (action: string) => void): void {
  switch (e.key) {
    case 'Escape':
      act('cancel');
      break;
    case 'F5':
      e.preventDefault();
      act('hold');
      break;
    case 'F6':
      e.preventDefault();
      act('pricecheck');
      break;
    case 'F7':
      e.preventDefault();
      act('quantity');
      break;
    case 'F8':
      e.preventDefault();
      act('discount');
      break;
    case 'F10':
      e.preventDefault();
      act('pay');
      break;
  }
}
