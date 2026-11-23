<script lang="ts">
  import SaleList from '#lib/sales/SaleList.svelte';

  const monthStart = new Date();
  monthStart.setDate(1);
  const iso = (d: Date) => d.toISOString().slice(0, 10);
</script>

<SaleList
  title="Sales history"
  subtitle="Browse past transactions — defaults to this month; widen the dates to go further back."
  defaultFrom={iso(monthStart)}
/>
