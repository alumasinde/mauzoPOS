package printing

import (
	"os"
	"strings"
	"testing"
)

func TestDocumentPrintersSaveAsPDF(t *testing.T) {
	for name, want := range map[string]bool{
		"Microsoft Print to PDF":        true,
		"Microsoft XPS Document Writer": true,
		"OneNote (Desktop)":             true,
		"Fax":                           true,
		"XP-58 (copy 1)":                false,
		"EPSON TM-T20II Receipt":        false,
	} {
		c := Config{Kind: KindWindows, Name: name}
		if got := c.SavesPDF(); got != want {
			t.Errorf("%q: SavesPDF = %v, want %v", name, got, want)
		}
	}
	if !(Config{Kind: KindPDF}).Configured() {
		t.Fatal("pdf kind needs no name")
	}
}

func TestSavePDF(t *testing.T) {
	t.Setenv("MAUZO_RECEIPTS_DIR", t.TempDir())
	path, err := SavePDF("receipt-12", []byte("%PDF-1.4\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(path, "receipt-12-") || !strings.HasSuffix(path, ".pdf") {
		t.Fatalf("path %s", path)
	}
	if b, _ := os.ReadFile(path); string(b) != "%PDF-1.4\n" {
		t.Fatal("content not written")
	}
}
