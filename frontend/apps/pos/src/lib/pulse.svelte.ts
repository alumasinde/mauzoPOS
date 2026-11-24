// Terminal pulse — device info, server status, layout, and the sync/heartbeat
// lifecycle that keeps the back office's online flag fresh and queued offline
// sales flowing. Owns the state so App.svelte stays under the line budget.
import { toast } from '@mauzo/ui';
import { desktop, type PayResult, type POSLayout, type Status, type TerminalInfo } from './desktop';
import { cart } from './cart.svelte';
import { mpesafeed } from './mpesafeed.svelte';

const bridge = desktop();
let info = $state<TerminalInfo | null>(null);
let status = $state<Status | null>(null);
let layout = $state<POSLayout | null>(null);

export const pulse = {
  get info() {
    return info;
  },
  set info(v) {
    info = v;
  },
  get status() {
    return status;
  },
  get layout() {
    return layout;
  },

  async refresh() {
    if (!bridge) return;
    status = await bridge.Status();
  },

  async loadLayout() {
    layout = await bridge!.Layout().catch(() => null);
  },

  // One beat: refresh last_seen server-side, then flush queued offline
  // sales when the till is back online.
  async heartbeat() {
    if (!bridge || !info?.activated) return;
    const online = await bridge.Heartbeat();
    if (online && (status?.pending_sales ?? 0) > 0) {
      const sent = await bridge.SyncPending().catch(() => 0);
      if (sent > 0) toast.success(`Uploaded ${sent} queued sale${sent === 1 ? '' : 's'}.`);
    }
    if (online) void mpesafeed.start(); // idempotent — reconnects when dropped
    void this.refresh();
  },

  async sync() {
    if (!bridge) return;
    const r = await bridge.Sync();
    if (!r.ok) {
      toast.error(r.error ?? 'Sync failed');
      return;
    }
    const sent = await bridge.SyncPending().catch(() => 0);
    toast.success(
      `Synced ${r.products} products, ${r.cashiers} cashiers.${sent ? ` Uploaded ${sent} queued sales.` : ''}`,
    );
    await this.refresh();
  },

  async hold(label: string) {
    if (!bridge || cart.count === 0) return;
    try {
      await bridge.Hold(label, cart.payload());
      await cart.clear();
      toast.success('Sale parked.');
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Could not hold sale');
    }
  },

  async paid(r: PayResult) {
    await cart.clear();
    toast.success(
      r.offline ? 'Sale queued — will upload on next sync.' : `Sale complete — receipt #${r.receipt_no}.`,
    );
    void this.refresh();
  },
};
