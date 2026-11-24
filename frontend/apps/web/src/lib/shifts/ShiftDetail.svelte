<script lang="ts">
  // Shift detail — Z/X report for the row, movements, variance review and
  // force-close (permission-gated).
  import { Badge, Button, Icon, Modal, toast } from '@mauzo/ui';
  import { formatDate, formatMoney } from '@mauzo/utils';
  import { ApiError } from '@mauzo/api-client';
  import { auth } from '#lib/auth.svelte.ts';
  import { shiftsApi, type Shift, type ShiftReport } from './api.ts';

  let {
    shift,
    onclose,
    onchanged,
  }: {
    shift: Shift;
    onclose: () => void;
    onchanged: () => void;
  } = $props();

  let report = $state<ShiftReport | null>(null);
  let reviewNote = $state('');
  let busy = $state('');

  const canCloseOthers = $derived(auth.can('shifts.shift.close_others'));
  const canReview = $derived(auth.can('shifts.variance.approve'));
  const varianceNum = $derived(Number(report?.shift.variance ?? 0));

  async function load() {
    try {
      report = await shiftsApi.report(shift.id);
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Could not load the shift');
    }
  }

  async function act(kind: 'approve' | 'flag' | 'force', done: string) {
    busy = kind;
    try {
      if (kind === 'force') {
        if (!confirm('Force-close this shift? No blind count — variance cannot be computed.')) {
          busy = '';
          return;
        }
        await shiftsApi.forceClose(shift.id, reviewNote || 'Force-closed from back office');
      } else {
        await shiftsApi.review(shift.id, kind === 'approve' ? 'approved' : 'flagged', reviewNote);
      }
      toast.success(done);
      reviewNote = '';
      await load();
      onchanged();
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Action failed');
    } finally {
      busy = '';
    }
  }

  load();
</script>

<Modal open title="Shift — {shift.terminal}" size="lg" {onclose}>
  {#if !report}
    <p class="muted">Loading…</p>
  {:else}
    <div class="meta">
      <div><span>Cashier</span><b>{report.shift.opened_by_name}</b></div>
      <div><span>Branch</span><b>{report.shift.branch}</b></div>
      <div><span>Opened</span><b>{formatDate(report.shift.opened_at)}</b></div>
      <div>
        <span>Status</span>
        <Badge tone={report.shift.status === 'open' ? 'success' : 'neutral'}>
          {report.shift.status}
        </Badge>
      </div>
      {#if report.shift.closed_at}
        <div><span>Closed</span><b>{formatDate(report.shift.closed_at)}</b></div>
        <div><span>Closed by</span><b>{report.shift.closed_by_name || '—'}</b></div>
      {/if}
    </div>

    <div class="grid">
      <div class="panel">
        <h4>Sales</h4>
        <div class="r total">
          <span>Total × {report.sales_count}</span><b>{formatMoney(report.sales_total)}</b>
        </div>
        {#each report.methods as m (m.method)}
          <div class="r"><span>{m.method} × {m.count}</span><span>{formatMoney(m.total)}</span></div>
        {/each}
      </div>
      <div class="panel">
        <h4>Cash drawer</h4>
        <div class="r"><span>Float</span><span>{formatMoney(report.shift.float)}</span></div>
        <div class="r"><span>Cash sales</span><span>{formatMoney(report.cash_sales)}</span></div>
        <div class="r"><span>Cash in</span><span>+{formatMoney(report.cash_in)}</span></div>
        <div class="r"><span>Cash out</span><span>−{formatMoney(report.cash_out)}</span></div>
        <div class="r"><span>Cash refunds</span><span>−{formatMoney(report.cash_refunded)}</span></div>
        <div class="r total"><span>Expected</span><b>{formatMoney(report.expected)}</b></div>
        {#if report.shift.declared !== null}
          <div class="r"><span>Counted</span><span>{formatMoney(report.shift.declared)}</span></div>
        {/if}
        {#if report.shift.variance !== null}
          <div class="r">
            <span>Variance</span>
            <Badge tone={varianceNum === 0 ? 'success' : varianceNum > 0 ? 'info' : 'danger'}>
              {varianceNum >= 0 ? '+' : ''}{formatMoney(report.shift.variance)}
            </Badge>
          </div>
        {/if}
      </div>
    </div>

    {#if report.movements.length > 0}
      <h4>Movements</h4>
      <div class="moves">
        {#each report.movements as mv (mv.id)}
          <div class="r">
            <span>
              {formatDate(mv.created_at)} · {mv.created_by_name}
              {mv.reason && mv.reason !== 'float' ? `· ${mv.reason}` : ''}
            </span>
            <Badge tone={mv.kind === 'cash_in' ? 'info' : mv.kind === 'cash_out' ? 'warning' : 'neutral'}>
              {mv.kind === 'drawer_open' ? 'drawer' : formatMoney(mv.amount)}
            </Badge>
          </div>
        {/each}
      </div>
    {/if}

    {#if report.shift.status === 'closed' && report.shift.variance_status !== 'pending'}
      <p class="muted">
        Variance {report.shift.variance_status}
        {report.shift.variance_note ? `— ${report.shift.variance_note}` : ''}
      </p>
    {/if}
  {/if}

  {#snippet footer()}
    {#if canCloseOthers && shift.status === 'open'}
      <Button variant="danger" loading={busy === 'force'} onclick={() => act('force', 'Shift force-closed')}>
        <Icon name="lock" size={14} /> Force close
      </Button>
    {/if}
    {#if canReview && shift.status === 'closed'}
      <input class="note" bind:value={reviewNote} placeholder="Review note" />
      <Button variant="secondary" loading={busy === 'flag'} onclick={() => act('flag', 'Variance flagged')}>
        Flag
      </Button>
      <Button
        variant="primary"
        loading={busy === 'approve'}
        onclick={() => act('approve', 'Variance approved')}
      >
        Approve
      </Button>
    {/if}
    <Button variant="ghost" onclick={onclose}>Close</Button>
  {/snippet}
</Modal>

<style>
  .muted {
    color: var(--color-text-2);
  }
  .meta {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: var(--space-3);
    margin-bottom: var(--space-4);
  }
  .meta span {
    display: block;
    font-size: var(--text-xs);
    color: var(--color-text-2);
  }
  .grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--space-3);
  }
  .panel {
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    padding: var(--space-3);
  }
  h4 {
    margin: 0 0 var(--space-2);
    font-size: var(--text-sm);
  }
  .r {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 4px 0;
    font-size: var(--text-sm);
  }
  .r.total {
    border-top: 1px dashed var(--color-border);
    margin-top: 4px;
    padding-top: 8px;
    font-weight: 600;
  }
  .moves {
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    padding: var(--space-2) var(--space-3);
    max-height: 220px;
    overflow-y: auto;
  }
  .note {
    flex: 1;
    min-width: 120px;
    padding: var(--space-2);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    font: inherit;
  }
</style>
