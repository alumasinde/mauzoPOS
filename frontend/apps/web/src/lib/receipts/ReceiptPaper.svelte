<script lang="ts">
  // Draws a rendered receipt (pkg/receiptfmt Output) as thermal paper. The
  // server already wrapped and aligned every text line to the paper's
  // character width, so this only reproduces the printer's fonts: tall,
  // wide and double characters are the normal glyphs scaled ×2.
  import type { ReceiptLine, ReceiptOutput } from '@mauzo/types';

  interface Props {
    output: ReceiptOutput;
    /** Object URL of the 1-bit logo (null hides logo lines). */
    logoUrl?: string | null;
    /** Highlight print-only area when printing from the browser. */
    printable?: boolean;
  }
  let { output, logoUrl = null, printable = false }: Props = $props();

  // A 58 mm roll prints 384 dots, 80 mm prints 576; the logo is ≤ 384 dots.
  const paperDots = $derived(output.paper === 58 ? 384 : 576);

  const range = (n = 1) => Array.from({ length: Math.max(1, n) }, (_, i) => i);

  function bars(l: ReceiptLine): { x: number; w: number }[] {
    const out: { x: number; w: number }[] = [];
    const s = l.bars ?? '';
    for (let i = 0; i < s.length; i++) {
      if (s[i] !== '1') continue;
      let j = i;
      while (s[j + 1] === '1') j++;
      out.push({ x: i, w: j - i + 1 });
      i = j;
    }
    return out;
  }
</script>

<div class="paper" class:printable style:--cols={output.width} role="img" aria-label="Receipt preview">
  {#each output.lines as l, i (i)}
    {#if l.k === 'text'}
      <div class="ln {l.s ?? ''}" class:b={l.b}><span>{l.t || ' '}</span></div>
    {:else if l.k === 'feed'}
      {#each range(l.n) as j (j)}<div class="ln">&nbsp;</div>{/each}
    {:else if l.k === 'logo'}
      <div class="media {l.a ?? 'center'}">
        {#if logoUrl}
          <img src={logoUrl} alt="Logo" class="logo" style:max-width="{(384 / paperDots) * 100}%" />
        {:else}
          <div class="ph logo-ph">Logo</div>
        {/if}
      </div>
    {:else if l.k === 'qr'}
      <div class="media {l.a ?? 'center'}">
        <div class="ph qr" title={l.d}>
          <span>QR</span>
          <small>{l.d}</small>
        </div>
      </div>
    {:else if l.k === 'barcode'}
      {@const bs = bars(l)}
      <div class="media {l.a ?? 'center'}">
        <figure class="barcode">
          <svg viewBox="0 0 {l.bars?.length ?? 1} 30" preserveAspectRatio="none" aria-hidden="true">
            {#each bs as b, j (j)}<rect x={b.x} y="0" width={b.w} height="30" />{/each}
          </svg>
          <figcaption>{l.d}</figcaption>
        </figure>
      </div>
    {/if}
  {/each}
</div>

<style>
  .paper {
    font-family: 'JetBrains Mono', 'Consolas', 'Courier New', monospace;
    font-size: 12.5px;
    line-height: 1.35;
    color: #111;
    background: #fffef9;
    width: calc(var(--cols) * 1ch + 28px);
    max-width: 100%;
    padding: 18px 14px 26px;
    box-shadow:
      0 1px 2px rgb(0 0 0 / 0.12),
      0 8px 24px rgb(0 0 0 / 0.16);
    border-radius: 2px;
    overflow-x: auto;
    /* torn bottom edge */
    mask:
      linear-gradient(#000 0 0) top / 100% calc(100% - 8px) no-repeat,
      conic-gradient(from -45deg at bottom, #0000, #000 1deg 89deg, #0000 90deg) bottom / 12px 8px repeat-x;
  }
  .ln {
    white-space: pre;
    height: 1.35em;
  }
  .ln span {
    display: inline-block;
    transform-origin: 0 0;
  }
  .b {
    font-weight: 700;
  }
  .tall {
    height: 2.7em;
  }
  .tall span {
    transform: scale(1, 2);
  }
  .wide span {
    transform: scale(2, 1);
  }
  .double {
    height: 2.7em;
  }
  .double span {
    transform: scale(2, 2);
  }
  .media {
    display: flex;
    margin: 4px 0;
  }
  .media.center {
    justify-content: center;
  }
  .media.right {
    justify-content: flex-end;
  }
  .logo {
    image-rendering: pixelated;
    width: auto;
    height: auto;
  }
  .ph {
    display: grid;
    place-items: center;
    border: 1px dashed #999;
    color: #666;
    font-size: 11px;
  }
  .logo-ph {
    width: 50%;
    height: 48px;
  }
  .qr {
    width: 96px;
    height: 96px;
    gap: 2px;
    text-align: center;
    padding: 4px;
    background: repeating-conic-gradient(#e8e8e8 0 25%, #fff 0 50%) 0 0 / 12px 12px;
  }
  .qr span {
    font-weight: 700;
    background: #fff;
    padding: 0 4px;
  }
  .qr small {
    font-size: 8px;
    word-break: break-all;
    background: #fff;
    max-height: 3.2em;
    overflow: hidden;
  }
  .barcode {
    margin: 0;
    width: 70%;
    text-align: center;
  }
  .barcode svg {
    width: 100%;
    height: 44px;
    display: block;
  }
  .barcode rect {
    fill: #111;
  }
  .barcode figcaption {
    font-size: 11px;
  }
  @media print {
    .paper.printable {
      box-shadow: none;
      mask: none;
      padding: 0;
      width: auto;
    }
  }
</style>
