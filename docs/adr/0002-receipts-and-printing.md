# ADR 0002 — Receipt layouts, issued receipts and printing

Status: accepted (Phase 8)

## Context

Every sale needs a receipt. The till must print it even when the server is
unreachable, the same layout must be previewable in the back office, and a
receipt handed to a customer must stay verifiable later: edits to a layout must
not change receipts already issued, and extra copies must be controlled
(a second receipt is a common fraud vector).

## Decision

1. **One renderer, shared.** `backend/pkg/receiptfmt` turns a *template*
   (versioned JSON: paper width + ordered blocks) and a *Doc* (business,
   branch, sale, items, payments, VAT, eTIMS) into an *Output*: lines already
   wrapped and column-aligned to the paper's character width (32 for 58 mm,
   48 or 42 for 80 mm). The API uses it for previews, snapshots, back-office
   copies and A4 data; the Windows POS links the same package and encodes
   Output to ESC/POS. A receipt therefore looks identical everywhere.

2. **Templates are data, validated.** Block types: logo, text (with
   `{{variables}}`), divider, fields, items, totals, payments, tax_summary,
   fiscal (eTIMS), qr, barcode, feed. Each block may have an `if` condition
   (flag such as `has.mpesa`, or "variable is non-empty", optionally negated).
   Unknown fields, variables or conditions are rejected. Three builtins ship:
   supermarket (80 mm), minimart (58 mm, Lipa na M-Pesa), restaurant (80 mm,
   tax invoice). The business profile's `receipt_template` picks the default.

3. **Versioned layouts, per branch.** Every save bumps `version` and appends
   to `receipt_template_versions`. `receipt_assignments` maps a branch (or
   0 = all branches) to a layout; with no assignment the profile builtin is
   used. A stored layout that fails to parse falls back to the builtin — a
   till must always print.

4. **Issued receipts are snapshots.** On sale completion the API renders
   the receipt once and stores the Output in `issued_receipts` (copy 0) with
   the template id + version (or builtin key). It is never re-rendered.
   Reprints are new rows (copy 1, 2, …) of the same snapshot with a
   `*** REPRINT n ***` banner at top and bottom, written in the same
   transaction as an `audit_logs` row (`receipt.reprinted`). Reprinting needs
   `pos.receipt.reprint` (sensitive) or, at the till, a supervisor grant.
   Banners (REPRINT, VOIDED, PROVISIONAL - OFFLINE, SAMPLE) are added by the
   renderer, not the template, so no layout can hide them.

5. **The till prints from a kit.** The sync bundle carries a *kit*: the
   branch layout, header details, the logo (already dithered to 1 bit on
   the server) and the print policy (auto-print, copies, open drawer). Online,
   the till prints the server's snapshot; offline it renders a
   **PROVISIONAL - OFFLINE** receipt from the local quote, and the server
   issues the real receipt when the sale uploads. Reprints require the
   server so every copy is counted.

6. **Transports.** ESC/POS (Epson-compatible) is sent RAW: to a printer
   installed in Windows through the spooler (USB/Bluetooth thermal printers
   with any driver, including "Generic / Text Only"), or to a network
   printer on TCP 9100. Text is reduced to ASCII (PC437) so accents never
   print as garbage. QR codes and Code 128 barcodes use the printer's native
   commands; previews draw the barcode and show a QR placeholder.

7. **A4 / PDF.** The back office shows an A4 tax invoice built from the same
   Doc; "Save as PDF" in the browser's print dialog produces the PDF. No
   server-side PDF library is added.

## Consequences

- Changing a layout reaches tills on their next sync; receipts already
  issued are unaffected.
- Totals on a provisional offline receipt come from the till's local quote
  and can differ from the server's if prices changed; it is clearly marked.
- Printing was verified to the Windows spooler API and over TCP, not on a
  physical thermal printer; the first install should run "Print test
  receipt" in the till's settings.
- eTIMS (Phase 10) only has to fill `Doc.Fiscal`; the fiscal block already
  renders when present.
