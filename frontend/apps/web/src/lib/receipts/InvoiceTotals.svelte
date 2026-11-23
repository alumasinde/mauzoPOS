<script lang="ts">
  // VAT breakdown, payments and totals of the A4 invoice.
  import type { ReceiptDoc } from '@mauzo/types';
  import { METHODS, money } from './invoice';

  let { doc }: { doc: ReceiptDoc } = $props();
</script>

<div class="sums">
  <div class="left">
    {#if doc.taxes?.length}
      <table class="vat">
        <thead><tr><th>VAT rate</th><th class="r">Taxable</th><th class="r">VAT</th></tr></thead>
        <tbody>
          {#each doc.taxes as t (t.rate)}
            <tr>
              <td>{Number(t.rate)}%</td><td class="r">{money(t.taxable)}</td><td class="r">{money(t.tax)}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
    <h3>Payment</h3>
    <ul>
      {#each doc.payments as p, i (i)}
        <li>
          {METHODS[p.method] ?? p.method}
          {money(p.amount)}{#if p.reference}
            · Ref {p.reference}{/if}
        </li>
      {/each}
    </ul>
  </div>
  <table class="totals">
    <tbody>
      <tr><td>Subtotal</td><td class="r">{money(doc.sale.subtotal)}</td></tr>
      {#if Number(doc.sale.discount ?? 0) > 0}
        <tr><td>Discount</td><td class="r">-{money(doc.sale.discount)}</td></tr>
      {/if}
      {#if Number(doc.sale.tax ?? 0) > 0}
        <tr>
          <td>VAT{doc.sale.prices_include_vat ? ' (included)' : ''}</td>
          <td class="r">{money(doc.sale.tax)}</td>
        </tr>
      {/if}
      <tr class="grand">
        <td>Total ({doc.sale.currency || 'KES'})</td><td class="r">{money(doc.sale.total)}</td>
      </tr>
      {#if Number(doc.sale.change ?? 0) > 0}
        <tr><td>Change</td><td class="r">{money(doc.sale.change)}</td></tr>
      {/if}
    </tbody>
  </table>
</div>

<style>
  .sums {
    display: flex;
    justify-content: space-between;
    gap: 10mm;
    margin-top: 6mm;
  }
  .left {
    flex: 1;
  }
  h3 {
    margin: 0 0 1mm;
    font-size: 9pt;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: #555;
  }
  table {
    border-collapse: collapse;
  }
  .vat {
    min-width: 70mm;
    margin-bottom: 4mm;
  }
  .vat th,
  .vat td {
    padding: 1mm 2mm;
    border-bottom: 1px solid #ddd;
    font-size: 9.5pt;
  }
  .vat th {
    text-align: left;
    color: #555;
  }
  ul {
    margin: 0;
    padding-left: 4mm;
  }
  .totals {
    width: 75mm;
  }
  .totals td {
    padding: 1.2mm 1mm;
  }
  .grand td {
    border-top: 2px solid #111;
    font-weight: 800;
    font-size: 12.5pt;
    padding-top: 2mm;
  }
  .r {
    text-align: right !important;
    white-space: nowrap;
  }
</style>
