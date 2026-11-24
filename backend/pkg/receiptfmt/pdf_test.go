package receiptfmt

import (
	"bytes"
	"os"
	"regexp"
	"strconv"
	"testing"
)

func TestEncodePDF(t *testing.T) {
	for _, key := range []string{"minimart", "supermarket"} {
		b, ok := BuiltinByKey(key)
		if !ok {
			t.Fatalf("builtin %s missing", key)
		}
		pdf := EncodePDF(Render(b.Template, SampleDoc()))
		if !bytes.HasPrefix(pdf, []byte("%PDF-1.4")) || !bytes.HasSuffix(pdf, []byte("%%EOF\n")) {
			t.Fatalf("%s: not a PDF envelope", key)
		}
		// startxref must point at the xref table.
		m := regexp.MustCompile(`startxref\n(\d+)`).FindSubmatch(pdf)
		if m == nil {
			t.Fatalf("%s: no startxref", key)
		}
		off, _ := strconv.Atoi(string(m[1]))
		if !bytes.HasPrefix(pdf[off:], []byte("xref")) {
			t.Fatalf("%s: startxref %d does not point at xref", key, off)
		}
		if bytes.Contains(pdf, []byte("{Y")) {
			t.Fatalf("%s: unflipped coordinate left in content", key)
		}
		if !bytes.Contains(pdf, []byte("TOTAL")) && !bytes.Contains(pdf, []byte("Total")) {
			t.Fatalf("%s: totals missing", key)
		}
		if dir := os.Getenv("RECEIPT_PDF_DIR"); dir != "" {
			_ = os.WriteFile(dir+"/"+key+".pdf", pdf, 0o600)
		}
	}
}

func TestPDFString(t *testing.T) {
	if got := pdfString(`a(b)\c`); got != "a\\(b\\)\\\\c" {
		t.Fatalf("escape: %q", got)
	}
}
