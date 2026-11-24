<script lang="ts">
  // Shift flow inside the checkout overlay: open (float), cash in/out,
  // blind-count close and the X/Z report view. Owns the report state so
  // Modals.svelte stays slim.
  import { untrack } from 'svelte';
  import { toast } from '@mauzo/ui';
  import ShiftModal from './ShiftModal.svelte';
  import ShiftReportModal from './ShiftReportModal.svelte';
  import { loadXReport } from '../shift.svelte';
  import type { ShiftReport } from '../desktop';

  let {
    mode,
    currency = 'KSh',
    onclose,
  }: {
    mode: 'shiftopen' | 'shiftcash' | 'shiftclose' | 'shiftreport';
    currency?: string;
    onclose: () => void;
  } = $props();

  let view = $state(untrack(() => mode));
  let report = $state<ShiftReport | null>(null);

  $effect(() => {
    view = mode;
    if (mode === 'shiftreport' && !report) {
      loadXReport()
        .then((r) => (report = r))
        .catch((e) => {
          toast.error(e instanceof Error ? e.message : 'No report');
          onclose();
        });
    }
  });
</script>

{#if view === 'shiftopen' || view === 'shiftcash'}
  <ShiftModal mode={view === 'shiftopen' ? 'open' : 'cash'} {currency} {onclose} />
{:else if view === 'shiftclose'}
  <ShiftModal mode="close" {currency} {onclose} onclosed={(r) => ((report = r), (view = 'shiftreport'))} />
{:else if view === 'shiftreport' && report}
  <ShiftReportModal {report} {currency} {onclose} />
{/if}
