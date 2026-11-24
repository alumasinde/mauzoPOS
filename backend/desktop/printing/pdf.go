package printing

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ReceiptsDir is where saved PDF receipts go: Documents\Mauzo Receipts
// (MAUZO_RECEIPTS_DIR overrides it, for support and tests).
func ReceiptsDir() (string, error) {
	if d := os.Getenv("MAUZO_RECEIPTS_DIR"); d != "" {
		return d, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Documents", "Mauzo Receipts"), nil
}

// SavePDF writes a receipt PDF named after name (e.g. the receipt number)
// and returns its path. A clock suffix keeps reprints from overwriting.
func SavePDF(name string, pdf []byte) (string, error) {
	dir, err := ReceiptsDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", fmt.Errorf("cannot create %s: %w", dir, err)
	}
	if name == "" {
		name = "receipt"
	}
	path := filepath.Join(dir, fmt.Sprintf("%s-%s.pdf", name, time.Now().Format("20060102-150405")))
	if err := os.WriteFile(path, pdf, 0o600); err != nil {
		return "", fmt.Errorf("cannot save receipt: %w", err)
	}
	return path, nil
}

// Open shows a saved file in its default app (the PDF viewer).
func Open(path string) error { return openFile(path) }
