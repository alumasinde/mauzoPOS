package receiptfmt

import (
	"bytes"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// PrintOptions control the printer job around the receipt.
type PrintOptions struct {
	// Logo is the tenant logo raster; logo lines are skipped when nil.
	Logo *Raster
	// Cut feeds and cuts the paper at the end (most thermal printers).
	Cut bool
	// OpenDrawer pulses the cash drawer kick (pin 2) before printing.
	OpenDrawer bool
	// FeedLines before the cut so the last line clears the cutter (default 4).
	FeedLines int
}

// ESC/POS command bytes (Epson-compatible; supported by the common Kenyan
// thermal printers: Epson TM, Xprinter, Rongta, Sunmi, Bixolon).
const (
	esc = 0x1B
	gs  = 0x1D
	lf  = 0x0A
)

// EncodeESCPOS renders o as an ESC/POS byte stream. Text is converted to
// ASCII (code page PC437 is selected; accents are stripped, other symbols
// replaced) so output never prints mojibake.
func EncodeESCPOS(o Output, opt PrintOptions) []byte {
	var b bytes.Buffer
	b.Write([]byte{esc, '@'})    // initialise
	b.Write([]byte{esc, 't', 0}) // code page PC437
	if opt.OpenDrawer {
		b.Write([]byte{esc, 'p', 0, 25, 250})
	}
	for _, l := range o.Lines {
		switch l.Kind {
		case LineText:
			setAlign(&b, l.Align)
			setStyle(&b, l.Bold, l.Size)
			b.WriteString(ASCII(strings.TrimRight(l.Text, " ")))
			b.WriteByte(lf)
			setStyle(&b, false, "")
		case LineFeed:
			b.Write([]byte{esc, 'd', clampByte(l.N, 1, 10)})
		case LineLogo:
			if opt.Logo != nil && opt.Logo.Width > 0 {
				setAlign(&b, or(l.Align, "center"))
				writeRaster(&b, *opt.Logo)
			}
		case LineQR:
			setAlign(&b, or(l.Align, "center"))
			writeQR(&b, l.Data, l.QR)
			b.WriteByte(lf)
		case LineBarcode:
			setAlign(&b, or(l.Align, "center"))
			writeCode128(&b, l.Data)
		}
	}
	setAlign(&b, "left")
	if opt.Cut {
		n := opt.FeedLines
		if n <= 0 {
			n = 4
		}
		b.Write([]byte{esc, 'd', clampByte(n, 1, 10)})
		b.Write([]byte{gs, 'V', 66, 0}) // feed to cut position, partial cut
	}
	return b.Bytes()
}

func setAlign(b *bytes.Buffer, a string) {
	n := byte(0)
	switch a {
	case "center":
		n = 1
	case "right":
		n = 2
	}
	b.Write([]byte{esc, 'a', n})
}

func setStyle(b *bytes.Buffer, bold bool, size string) {
	e := byte(0)
	if bold {
		e = 1
	}
	b.Write([]byte{esc, 'E', e})
	var s byte
	switch size {
	case "tall":
		s = 0x01
	case "wide":
		s = 0x10
	case "double":
		s = 0x11
	}
	b.Write([]byte{gs, '!', s})
}

// writeQR uses the printer's native QR generator (GS ( k, model 2, EC level M).
func writeQR(b *bytes.Buffer, data string, size int) {
	d := []byte(ASCII(data))
	if len(d) > 700 {
		d = d[:700]
	}
	fn := func(params ...byte) {
		n := len(params) + 1                                            // pL/pH count cn plus fn and its parameters (params[0] is fn)
		b.Write([]byte{gs, '(', 'k', byte(n % 256), byte(n / 256), 49}) //nolint:gosec // length is masked to a byte
		b.Write(params)
	}
	fn(65, 50, 0)                         // model 2
	fn(67, clampByte(size, 1, 16))        // module size
	fn(69, 49)                            // error correction M
	store := append([]byte{80, 48}, d...) // store data
	fn(store...)
	fn(81, 48) // print
}

// writeCode128 prints a Code 128 (set B) barcode with its text below.
func writeCode128(b *bytes.Buffer, data string) {
	d := []byte(ASCII(data))
	if len(d) > 60 {
		d = d[:60]
	}
	b.Write([]byte{gs, 'h', 60}) // height in dots
	b.Write([]byte{gs, 'w', 2})  // module width
	b.Write([]byte{gs, 'H', 2})  // human-readable text below
	payload := append([]byte("{B"), d...)
	b.Write([]byte{gs, 'k', 73, byte(len(payload))}) //nolint:gosec // payload ≤ 64 bytes by construction
	b.Write(payload)
	b.WriteByte(lf)
}

// writeRaster prints a 1-bit image with GS v 0.
func writeRaster(b *bytes.Buffer, r Raster) {
	bpr := r.BytesPerRow()
	// ESC/POS little-endian byte pairs; values are masked.
	b.Write([]byte{gs, 'v', '0', 0, byte(bpr % 256), byte(bpr / 256), byte(r.Height % 256), byte(r.Height / 256)}) //nolint:gosec
	b.Write(r.Bits)
	b.WriteByte(lf)
}

// clampByte bounds n to [lo, hi] (both ≤ 255) and returns it as a byte.
func clampByte(n, lo, hi int) byte {
	return byte(clamp(n, lo, hi)) //nolint:gosec // bounded to 0–255 by clamp
}

func clamp(n, lo, hi int) int {
	if n < lo {
		return lo
	}
	if n > hi {
		return hi
	}
	return n
}

// ASCII converts text for a thermal printer: accents are stripped
// (é → e), common typographic symbols mapped, anything else becomes "?".
func ASCII(s string) string {
	repl := map[rune]string{
		'–': "-", '—': "-", '‘': "'", '’': "'", '“': "\"", '”': "\"",
		'…': "...", '·': "-", '•': "*", '×': "x", '€': "EUR", '£': "GBP",
		' ': " ",
	}
	var b strings.Builder
	for _, r := range norm.NFD.String(s) {
		switch {
		case r < 0x80 && (r >= 0x20 || r == '\n'):
			b.WriteRune(r)
		case unicode.Is(unicode.Mn, r):
			// combining accent from NFD: drop
		default:
			if v, ok := repl[r]; ok {
				b.WriteString(v)
			} else {
				b.WriteByte('?')
			}
		}
	}
	return b.String()
}
