// The payment methods this shop has switched on (setting `payments.methods`,
// synced to the till). The action bar and the payment drawer both read it, so
// a method that is off never shows up as a button.
import { desktop } from '../desktop';
import type { Method } from './tender.svelte';

export interface MethodInfo {
  label: string;
  hint: string;
  icon: string;
}

export const METHOD_INFO: Record<Method, MethodInfo> = {
  cash: { label: 'Cash', hint: 'Cash tender', icon: 'banknote' },
  mpesa: { label: 'M-Pesa', hint: 'Till · prompt · code', icon: 'phone' },
  card: { label: 'Card', hint: 'Card machine', icon: 'credit-card' },
  bank: { label: 'Bank', hint: 'Bank transfer', icon: 'landmark' },
  credit: { label: 'On account', hint: 'Customer credit', icon: 'book' },
};

const ALL: Method[] = ['cash', 'mpesa', 'card', 'bank', 'credit'];
const FALLBACK: Method[] = ['cash', 'mpesa', 'card'];

class PayMethods {
  list = $state<Method[]>(FALLBACK);

  /** Re-reads the synced setting; keeps the last good list on failure. */
  async load(): Promise<Method[]> {
    const got = await desktop()
      ?.PaymentMethods()
      .catch(() => null);
    const valid = (got ?? []).filter((m): m is Method => ALL.includes(m as Method));
    if (valid.length) this.list = valid;
    return this.list;
  }

  /** The wanted method if it is on, otherwise the first one that is. */
  pick(wanted: string): Method {
    return this.list.includes(wanted as Method) ? (wanted as Method) : (this.list[0] ?? 'cash');
  }
}

export const paymethods = new PayMethods();
