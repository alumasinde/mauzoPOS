<script lang="ts">
  // A4 tax invoice / receipt of one sale, styled for printing.
  import type { ReceiptDoc } from '@mauzo/types';
  import InvoiceTotals from './InvoiceTotals.svelte';
  import { money, qty, when } from './invoice';

  let { doc, status }: { doc: ReceiptDoc; status: string } = $props();
</script>

<article class="a4">
  {#if status === 'voided'}<div class="stamp">VOIDED</div>{/if}
  <header>
    <div>
      <h1>{doc.business.name}</h1>
      <p>
        {#if doc.branch.name}{doc.branch.name}<br />{/if}
        {#if doc.branch.location || doc.business.address}{doc.branch.location || doc.business.address}<br
          />{/if}
        {#if doc.branch.phone || doc.business.phone}Tel: {doc.branch.phone || doc.business.phone}<br />{/if}
        {#if doc.business.email}{doc.business.email}<br />{/if}
        {#if doc.business.kra_pin}<strong>KRA PIN: {doc.business.kra_pin}</strong>{/if}
      </p>
    </div>
    <div class="meta">
      <h2>{doc.business.vat_registered ? 'TAX INVOICE' : 'RECEIPT'}</h2>
      <dl>
        <dt>Number</dt>
        <dd>{doc.sale.receipt_no}</dd>
        <dt>Date</dt>
        <dd>{when(doc.sale.sold_at)}</dd>
        {#if doc.sale.cashier}<dt>Served by</dt>
          <dd>{doc.sale.cashier}</dd>{/if}
        {#if doc.sale.terminal}<dt>Till</dt>
          <dd>{doc.sale.terminal}</dd>{/if}
      </dl>
    </div>
  </header>

  {#if doc.sale.customer}
    <section class="bill">
      <h3>Bill to</h3>
      <p>
        {doc.sale.customer}
        {#if doc.sale.customer_pin}<br />PIN: {doc.sale.customer_pin}{/if}
      </p>
    </section>
  {/if}

  <table class="lines">
    <thead>
      <tr>
        <th>#</th><th>Description</th><th class="r">Qty</th><th class="r">Unit price</th>
        <th class="r">Discount</th><th class="r">VAT %</th><th class="r">Amount</th>
      </tr>
    </thead>
    <tbody>
      {#each doc.items as it, i (i)}
        <tr>
          <td>{i + 1}</td>
          <td>
            {it.name}{#if it.sku}<small> · {it.sku}</small>{/if}
          </td>
          <td class="r">{qty(it.qty)}</td>
          <td class="r">{money(it.unit_price)}</td>
          <td class="r">{Number(it.discount ?? 0) > 0 ? money(it.discount) : ''}</td>
          <td class="r">{it.tax_rate != null ? `${Number(it.tax_rate)}%` : '—'}</td>
          <td class="r">{money(it.total)}</td>
        </tr>
      {/each}
    </tbody>
  </table>

  <InvoiceTotals {doc} />

  {#if doc.fiscal}
    <section class="fiscal">
      <h3>KRA eTIMS</h3>
      <p>
        {#if doc.fiscal.cu_serial}CU serial: {doc.fiscal.cu_serial}<br />{/if}
        {#if doc.fiscal.invoice_no}CU invoice no: {doc.fiscal.invoice_no}<br />{/if}
        {#if doc.fiscal.signature}Signature: {doc.fiscal.signature}{/if}
      </p>
    </section>
  {/if}

  <footer>
    {#if doc.footer}<p>{doc.footer}</p>{/if}
    <p class="muted">Ref {doc.sale.ref}</p>
  </footer>
</article>

<style>
  .a4 {
    position: relative;
    width: 210mm;
    max-width: 100%;
    min-height: 260mm;
    margin: 0 auto;
    padding: 16mm 15mm;
    background: #fff;
    color: #111;
    box-shadow: var(--shadow-md);
    font-size: 10.5pt;
    line-height: 1.45;
  }
  header {
    display: flex;
    justify-content: space-between;
    gap: 10mm;
    padding-bottom: 6mm;
    border-bottom: 2px solid #111;
  }
  h1 {
    margin: 0 0 2mm;
    font-size: 18pt;
  }
  h2 {
    margin: 0 0 2mm;
    font-size: 15pt;
    letter-spacing: 0.06em;
    text-align: right;
  }
  h3 {
    margin: 0 0 1mm;
    font-size: 9pt;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: #555;
  }
  p {
    margin: 0;
  }
  .meta dl {
    display: grid;
    grid-template-columns: auto auto;
    gap: 0 4mm;
    margin: 0;
    justify-content: end;
  }
  .meta dt {
    color: #555;
  }
  .meta dd {
    margin: 0;
    text-align: right;
    font-weight: 600;
  }
  .bill {
    margin-top: 6mm;
  }
  .lines {
    width: 100%;
    border-collapse: collapse;
    margin-top: 6mm;
  }
  .lines th {
    text-align: left;
    font-size: 8.5pt;
    text-transform: uppercase;
    color: #555;
    border-bottom: 1px solid #111;
    padding: 1.5mm 1mm;
  }
  .lines td {
    padding: 1.6mm 1mm;
    border-bottom: 1px solid #ddd;
    vertical-align: top;
  }
  .lines small,
  .muted {
    color: #777;
  }
  .r {
    text-align: right !important;
    white-space: nowrap;
  }
  .fiscal {
    margin-top: 6mm;
    padding: 3mm 4mm;
    border: 1px solid #111;
  }
  .fiscal p {
    font-size: 9pt;
    word-break: break-all;
  }
  footer {
    margin-top: 10mm;
    padding-top: 3mm;
    border-top: 1px solid #ddd;
    text-align: center;
    font-size: 9pt;
  }
  .stamp {
    position: absolute;
    top: 70mm;
    left: 50%;
    transform: translateX(-50%) rotate(-18deg);
    font-size: 48pt;
    font-weight: 900;
    color: rgb(200 0 0 / 0.25);
    border: 6px solid rgb(200 0 0 / 0.25);
    padding: 2mm 8mm;
    pointer-events: none;
  }
  @media print {
    .a4 {
      box-shadow: none;
      width: 210mm;
      min-height: auto;
    }
  }
</style>
