//go:build !windows

package printing

import "errors"

var errNoSpooler = errors.New("installed printers are only available on Windows")

func spoolRaw(string, string, []byte) error { return errNoSpooler }

func probeSpooler(string) error { return errNoSpooler }

func listPrinters() ([]string, error) { return []string{}, nil }

func openFile(string) error { return nil }
