import type { ReceiptBlock, ReceiptBlockType } from '@mauzo/types';

/** What the designer offers for each block type. */
export interface BlockKind {
  type: ReceiptBlockType;
  label: string;
  hint: string;
  icon: string;
  make: () => ReceiptBlock;
}

export const BLOCKS: BlockKind[] = [
  {
    type: 'logo',
    label: 'Logo',
    hint: 'Your uploaded logo, printed in black and white.',
    icon: 'store',
    make: () => ({ type: 'logo', if: 'has.logo' }),
  },
  {
    type: 'text',
    label: 'Text',
    hint: 'Any text. Insert {{variables}} such as the business name.',
    icon: 'file',
    make: () => ({ type: 'text', text: '{{business.name}}', align: 'center' }),
  },
  {
    type: 'divider',
    label: 'Divider',
    hint: 'A full-width line.',
    icon: 'arrows',
    make: () => ({ type: 'divider' }),
  },
  {
    type: 'fields',
    label: 'Details',
    hint: 'Label … value rows: receipt number, date, cashier.',
    icon: 'clipboard',
    make: () => ({
      type: 'fields',
      fields: [
        { label: 'Receipt', value: '{{sale.receipt_no}}' },
        { label: 'Date', value: '{{sale.datetime}}' },
        { label: 'Served by', value: '{{sale.cashier}}' },
      ],
    }),
  },
  {
    type: 'items',
    label: 'Items',
    hint: 'The products sold.',
    icon: 'box',
    make: () => ({ type: 'items', layout: 'detailed' }),
  },
  {
    type: 'totals',
    label: 'Totals',
    hint: 'Subtotal, discount, VAT and the TOTAL.',
    icon: 'calculator',
    make: () => ({ type: 'totals' }),
  },
  {
    type: 'payments',
    label: 'Payments',
    hint: 'How the customer paid, M-Pesa references and change.',
    icon: 'wallet',
    make: () => ({ type: 'payments' }),
  },
  {
    type: 'tax_summary',
    label: 'VAT summary',
    hint: 'Taxable amount and VAT per rate.',
    icon: 'percent',
    make: () => ({ type: 'tax_summary', if: 'has.tax' }),
  },
  {
    type: 'fiscal',
    label: 'eTIMS block',
    hint: 'KRA control unit details. Prints only once the sale is signed.',
    icon: 'shield',
    make: () => ({ type: 'fiscal', if: 'has.fiscal' }),
  },
  {
    type: 'barcode',
    label: 'Barcode',
    hint: 'A scannable code, e.g. the receipt number for quick returns.',
    icon: 'chart-bars',
    make: () => ({ type: 'barcode', data: '{{sale.receipt_no}}' }),
  },
  {
    type: 'qr',
    label: 'QR code',
    hint: 'A QR code of a link or text (printed by the printer).',
    icon: 'sparkle',
    make: () => ({ type: 'qr', data: 'https://', qr_size: 6 }),
  },
  {
    type: 'feed',
    label: 'Blank space',
    hint: 'Empty lines.',
    icon: 'layers',
    make: () => ({ type: 'feed', lines: 1 }),
  },
];

export const kindOf = (t: ReceiptBlockType): BlockKind => BLOCKS.find((b) => b.type === t) ?? BLOCKS[1]!;

/** A one-line summary of a block for the layout list. */
export function summary(b: ReceiptBlock): string {
  switch (b.type) {
    case 'text':
      return (b.text ?? '').split('\n')[0] ?? '';
    case 'fields':
      return (b.fields ?? []).map((f) => f.label).join(', ');
    case 'items':
      return b.layout === 'compact' ? 'One line per item' : 'Name, then qty × price';
    case 'divider':
      return (b.char ?? '-').repeat(12);
    case 'feed':
      return `${b.lines ?? 1} line${(b.lines ?? 1) === 1 ? '' : 's'}`;
    case 'qr':
    case 'barcode':
      return b.data ?? '';
    default:
      return kindOf(b.type).hint;
  }
}

/** Block types whose text can be aligned / styled. */
export const STYLED: ReceiptBlockType[] = ['text'];
export const ALIGNED: ReceiptBlockType[] = ['text', 'logo', 'qr', 'barcode'];
