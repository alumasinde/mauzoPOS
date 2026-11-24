// M-Pesa collection feed — keeps the till's unallocated-payments list fresh
// from the SSE pump (Go side) plus a hydration call on reconnect. Feed events
// also surface a toast so a walk-in payment is visible the moment it lands.
import { toast } from '@mauzo/ui';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { desktop, type MpesaFeedEvent, type MpesaTxn } from './desktop';
import { formatMoney } from './money';

const bridge = desktop();
let items = $state<MpesaTxn[]>([]);
let listening = false;
// Waiting STK prompts: notified the moment their result arrives on the feed.
let stkWatchers: ((t: MpesaTxn) => void)[] = [];

export const mpesafeed = {
  get items() {
    return items;
  },
  get count() {
    return items.length;
  },

  /** Open the SSE pump and hydrate the unallocated list. Safe to call again —
   * the Go side ignores duplicate starts. */
  async start() {
    if (!bridge) return;
    try {
      await bridge.MpesaFeedStart();
    } catch {
      return; // not activated / offline — retry on next heartbeat
    }
    await this.refresh();
    if (listening) return;
    listening = true;
    EventsOn('mpesa:feed', (raw: unknown) => {
      try {
        const ev = typeof raw === 'string' ? (JSON.parse(raw) as MpesaFeedEvent) : (raw as MpesaFeedEvent);
        this.onEvent(ev);
      } catch {
        /* malformed event — ignore */
      }
    });
  },

  stop() {
    void bridge?.MpesaFeedStop();
  },

  /** Re-read the unallocated queue (reconnect hydration). */
  async refresh() {
    if (!bridge) return;
    try {
      const res = await bridge.MpesaRecent();
      items = res.items ?? [];
    } catch {
      /* offline — keep last known list */
    }
  },

  /** Calls fn with every prompt result for this branch until unsubscribed. */
  onStk(fn: (t: MpesaTxn) => void): () => void {
    stkWatchers = [...stkWatchers, fn];
    return () => {
      stkWatchers = stkWatchers.filter((w) => w !== fn);
    };
  },

  onEvent(ev: MpesaFeedEvent) {
    if (!ev?.txn) return;
    if (ev.kind === 'stk') {
      for (const fn of stkWatchers) fn(ev.txn);
      return;
    }
    if (ev.kind === 'collection') {
      const label = ev.txn.payer_name || ev.txn.phone;
      toast.success(`M-Pesa KSh ${formatMoney(ev.txn.amount)} from ${label}`);
      items = [ev.txn, ...items.filter((t) => t.public_id !== ev.txn.public_id)];
    } else if (ev.kind === 'reversal') {
      items = items.filter((t) => t.public_id !== ev.txn.public_id);
      toast.info(`M-Pesa ${ev.txn.trans_id ?? 'payment'} ${ev.txn.status}.`);
    }
  },

  /** Drop a transaction after it funds a sale (consume or allocate). */
  drop(publicID: string) {
    items = items.filter((t) => t.public_id !== publicID);
  },

  /** Drop by M-Pesa code — what a payment row cites. */
  dropByTrans(transID: string) {
    items = items.filter((t) => t.trans_id !== transID);
  },
};
