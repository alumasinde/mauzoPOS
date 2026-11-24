package receiptfmt

import (
	"bytes"
	"fmt"
	"strings"
)

// EncodePDF draws a laid-out receipt as a one-page PDF, receipt-width, in
// Courier so the columns line up exactly as on paper. It is for tills with no
// thermal printer ("save as PDF") and for e-mailing a copy; ESC/POS stays the
// format for real printers. Logos and QR codes are not drawn (a QR payload is
// printed as text); Code 128 barcodes are drawn as bars.
func EncodePDF(o Output) []byte {
	const margin = 10.0
	paper := o.Paper
	if paper <= 0 {
		paper = 80
	}
	width := o.Width
	if width <= 0 {
		width = 48
	}
	pageW := float64(paper) * 72 / 25.4
	ops, y := pdfLayout(o.Lines, margin, pageW-2*margin, width)
	pageH := y + 2*margin

	// Flip the top-down positions into PDF's bottom-up coordinates.
	var content bytes.Buffer
	for _, op := range ops {
		content.WriteString(flipY(op, pageH-margin))
		content.WriteByte('\n')
	}
	return pdfDocument(pageW, pageH, content.Bytes())
}

// pdfLayout turns receipt lines into drawing operators with top-down y
// markers, returning them and the height used.
func pdfLayout(lines []Line, margin, usable float64, width int) ([]string, float64) {
	fs := usable / (float64(width) * 0.6) // Courier advance = 0.6 em
	lead := fs * 1.3
	var ops []string
	y := 0.0 // distance from the top, grows downwards; flipped when emitting
	for _, l := range lines {
		switch l.Kind {
		case LineText:
			size, scale, rowH := fs, 100.0, lead
			switch l.Size {
			case "tall":
				size, scale, rowH = fs*2, 50, lead*2
			case "wide":
				scale = 200
			case "double":
				size, rowH = fs*2, lead*2
			}
			y += rowH
			font := "F1"
			if l.Bold {
				font = "F2"
			}
			text := strings.TrimRight(l.Text, " ")
			adv := float64(len([]rune(text))) * size * 0.6 * scale / 100
			x := margin
			switch l.Align {
			case "center":
				x = margin + (usable-adv)/2
			case "right":
				x = margin + usable - adv
			}
			ops = append(ops, fmt.Sprintf("BT /%s %.2f Tf %.0f Tz %.2f {Y%.2f} Td (%s) Tj ET",
				font, size, scale, x, y-rowH*0.25, pdfString(text)))
		case LineBarcode:
			if l.Bars == "" {
				break
			}
			mod := min(usable/float64(len(l.Bars)), 1.4)
			x := margin + (usable-mod*float64(len(l.Bars)))/2
			h := 34.0
			y += h + 4
			for i, c := range l.Bars {
				if c == '1' {
					ops = append(ops, fmt.Sprintf("%.2f {Y%.2f} %.2f %.2f re f", x+float64(i)*mod, y, mod, h))
				}
			}
			y += lead
			ops = append(ops, centred(l.Data, fs, lead, margin, usable, y))
		case LineQR:
			y += lead
			ops = append(ops, centred(l.Data, fs, lead, margin, usable, y))
		case LineFeed:
			y += lead * float64(max(l.N, 1))
		}
	}
	return ops, y
}

// pdfDocument wraps one page's content stream in a minimal PDF 1.4 file
// with the two Courier fonts.
func pdfDocument(pageW, pageH float64, content []byte) []byte {
	objs := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %.2f %.2f] "+
			"/Resources << /Font << /F1 5 0 R /F2 6 0 R >> >> /Contents 4 0 R >>", pageW, pageH),
		fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", len(content), content),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Courier /Encoding /WinAnsiEncoding >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Courier-Bold /Encoding /WinAnsiEncoding >>",
	}
	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objs))
	for i, body := range objs {
		offsets[i] = out.Len()
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", i+1, body)
	}
	xref := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n0000000000 65535 f \n", len(objs)+1)
	for _, off := range offsets {
		fmt.Fprintf(&out, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objs)+1, xref)
	return out.Bytes()
}

func centred(s string, fs, lead, margin, usable, y float64) string {
	adv := float64(len([]rune(s))) * fs * 0.6
	return fmt.Sprintf("BT /F1 %.2f Tf 100 Tz %.2f {Y%.2f} Td (%s) Tj ET",
		fs, margin+(usable-adv)/2, y-lead*0.25, pdfString(s))
}

// flipY replaces each {Y<top-down>} marker with the PDF y coordinate.
func flipY(op string, top float64) string {
	for {
		i := strings.Index(op, "{Y")
		if i < 0 {
			return op
		}
		j := strings.IndexByte(op[i:], '}')
		var v float64
		_, _ = fmt.Sscanf(op[i+2:i+j], "%g", &v)
		op = op[:i] + fmt.Sprintf("%.2f", top-v) + op[i+j+1:]
	}
}

// pdfString escapes a literal string; characters outside Latin-1 become '?'.
func pdfString(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '(' || r == ')' || r == '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case r < 32:
			b.WriteByte(' ')
		case r < 128:
			b.WriteRune(r)
		case r < 256:
			fmt.Fprintf(&b, "\\%03o", r)
		default:
			b.WriteByte('?')
		}
	}
	return b.String()
}
