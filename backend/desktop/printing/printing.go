// Package printing sends ESC/POS jobs to the till's receipt printer. Two
// transports cover the hardware Kenyan shops use:
//
//   - windows: a printer installed in Windows (USB or Bluetooth thermal
//     printers with their vendor or "Generic / Text Only" driver). The job is
//     written RAW through the print spooler, so the driver never re-renders it.
//   - network: an Ethernet/Wi-Fi printer listening on TCP 9100 (raw port).
//
// A "pdf" transport saves each receipt as a PDF file and opens it — for tills
// with no thermal printer. Windows' own document printers (Microsoft Print to
// PDF, XPS Document Writer, OneNote, Fax) cannot take raw ESC/POS: sent RAW
// they produce an empty file, so receipts for them are saved as PDF too.
//
// A "file" transport appends jobs to a file for development and support.
package printing

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// Kinds of printer connection.
const (
	KindNone    = "none"
	KindWindows = "windows"
	KindNetwork = "network"
	KindFile    = "file"
	KindPDF     = "pdf"
)

// IsDocumentPrinter reports whether an installed Windows printer is a virtual
// document printer that writes files rather than paper.
func IsDocumentPrinter(name string) bool {
	n := strings.ToLower(name)
	for _, s := range []string{"pdf", "xps", "onenote", "fax", "document writer"} {
		if strings.Contains(n, s) {
			return true
		}
	}
	return false
}

// SavesPDF reports whether receipts for c are saved as PDF files.
func (c Config) SavesPDF() bool {
	return c.Kind == KindPDF || (c.Kind == KindWindows && IsDocumentPrinter(c.Name))
}

// Config is the terminal's printer connection.
type Config struct {
	Kind string `json:"kind"`
	Name string `json:"name,omitempty"` // Windows printer name, or file path for KindFile
	Host string `json:"host,omitempty"`
	Port int    `json:"port,omitempty"` // default 9100
}

// ErrNotConfigured means no printer is set up on this till.
var ErrNotConfigured = errors.New("no receipt printer is configured")

// Configured reports whether c points at a printer.
func (c Config) Configured() bool {
	switch c.Kind {
	case KindPDF:
		return true
	case KindWindows, KindFile:
		return strings.TrimSpace(c.Name) != ""
	case KindNetwork:
		return strings.TrimSpace(c.Host) != ""
	}
	return false
}

func (c Config) addr() string {
	port := c.Port
	if port <= 0 {
		port = 9100
	}
	return net.JoinHostPort(strings.TrimSpace(c.Host), strconv.Itoa(port))
}

// Send writes one raw job.
func Send(c Config, job []byte) error {
	if !c.Configured() {
		return ErrNotConfigured
	}
	if len(job) == 0 {
		return nil
	}
	switch c.Kind {
	case KindNetwork:
		conn, err := dial(c.addr(), 4*time.Second)
		if err != nil {
			return fmt.Errorf("printer %s unreachable: %w", c.addr(), err)
		}
		defer func() { _ = conn.Close() }()
		_ = conn.SetWriteDeadline(time.Now().Add(15 * time.Second))
		if _, err := conn.Write(job); err != nil {
			return fmt.Errorf("printer %s: %w", c.addr(), err)
		}
		return nil
	case KindWindows:
		return spoolRaw(c.Name, "Mauzo receipt", job)
	case KindFile:
		//nolint:gosec // the path is chosen by the terminal operator in settings
		f, err := os.OpenFile(c.Name, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()
		_, err = f.Write(job)
		return err
	}
	return ErrNotConfigured
}

// Probe reports whether the printer can be reached right now ("ready",
// "offline" or "not_configured"). It never prints.
func Probe(c Config) string {
	if !c.Configured() {
		return "not_configured"
	}
	switch c.Kind {
	case KindNetwork:
		conn, err := dial(c.addr(), 1500*time.Millisecond)
		if err != nil {
			return "offline"
		}
		_ = conn.Close()
		return "ready"
	case KindWindows:
		if err := probeSpooler(c.Name); err != nil {
			return "offline"
		}
		return "ready"
	}
	return "ready"
}

func dial(addr string, timeout time.Duration) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	var d net.Dialer
	return d.DialContext(ctx, "tcp", addr)
}

// List returns the printers installed in Windows (empty elsewhere).
func List() ([]string, error) { return listPrinters() }
