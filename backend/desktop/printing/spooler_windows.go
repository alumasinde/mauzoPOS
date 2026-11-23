//go:build windows

// The winspool calls below pass Go pointers to Win32 through syscall; each
// pointer refers to memory that outlives the call (gosec G103 reviewed).

package printing

import (
	"errors"
	"fmt"
	"syscall"
	"unsafe"
)

var (
	winspool         = syscall.NewLazyDLL("winspool.drv")
	procOpenPrinter  = winspool.NewProc("OpenPrinterW")
	procClosePrinter = winspool.NewProc("ClosePrinter")
	procStartDoc     = winspool.NewProc("StartDocPrinterW")
	procEndDoc       = winspool.NewProc("EndDocPrinter")
	procStartPage    = winspool.NewProc("StartPagePrinter")
	procEndPage      = winspool.NewProc("EndPagePrinter")
	procWritePrinter = winspool.NewProc("WritePrinter")
	procEnumPrinters = winspool.NewProc("EnumPrintersW")

	errShortWrite = errors.New("print spooler accepted only part of the job")
)

const (
	printerEnumLocal       = 0x00000002
	printerEnumConnections = 0x00000004
)

// docInfo1 mirrors DOC_INFO_1.
type docInfo1 struct {
	docName    *uint16
	outputFile *uint16
	datatype   *uint16
}

// printerInfo4 mirrors PRINTER_INFO_4.
type printerInfo4 struct {
	printerName *uint16
	serverName  *uint16
	attributes  uint32
}

func openPrinter(name string) (syscall.Handle, error) {
	p, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return 0, err
	}
	var h syscall.Handle
	//nolint:gosec // G103: Win32 FFI, pointers valid for the call
	r, _, e := procOpenPrinter.Call(uintptr(unsafe.Pointer(p)), uintptr(unsafe.Pointer(&h)), 0)
	if r == 0 {
		return 0, fmt.Errorf("open printer %q: %w", name, e)
	}
	return h, nil
}

func closePrinter(h syscall.Handle) { _, _, _ = procClosePrinter.Call(uintptr(h)) }

// spoolRaw sends job to an installed printer as a RAW document, bypassing the
// driver's rendering (the bytes are already ESC/POS).
func spoolRaw(printer, title string, job []byte) error {
	h, err := openPrinter(printer)
	if err != nil {
		return err
	}
	defer closePrinter(h)
	name, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		return err
	}
	raw, err := syscall.UTF16PtrFromString("RAW")
	if err != nil {
		return err
	}
	di := docInfo1{docName: name, datatype: raw}
	//nolint:gosec // G103: Win32 FFI, pointers valid for the call
	if r, _, e := procStartDoc.Call(uintptr(h), 1, uintptr(unsafe.Pointer(&di))); r == 0 {
		return fmt.Errorf("start document: %w", e)
	}
	defer func() { _, _, _ = procEndDoc.Call(uintptr(h)) }()
	if r, _, e := procStartPage.Call(uintptr(h)); r == 0 {
		return fmt.Errorf("start page: %w", e)
	}
	var written uint32
	//nolint:gosec // G103: Win32 FFI, pointers valid for the call
	r, _, e := procWritePrinter.Call(uintptr(h), uintptr(unsafe.Pointer(&job[0])), uintptr(len(job)),
		uintptr(unsafe.Pointer(&written)))
	_, _, _ = procEndPage.Call(uintptr(h))
	if r == 0 {
		return fmt.Errorf("write to printer: %w", e)
	}
	if int(written) != len(job) {
		return errShortWrite
	}
	return nil
}

func probeSpooler(printer string) error {
	h, err := openPrinter(printer)
	if err != nil {
		return err
	}
	closePrinter(h)
	return nil
}

// listPrinters enumerates local and connected printers (PRINTER_INFO_4).
func listPrinters() ([]string, error) {
	flags := uintptr(printerEnumLocal | printerEnumConnections)
	var needed, returned uint32
	// Win32 EnumPrinters requires out-pointer marshalling.
	_, _, _ = procEnumPrinters.Call(flags, 0, 4, 0, 0,
		uintptr(unsafe.Pointer(&needed)), uintptr(unsafe.Pointer(&returned))) //nolint:gosec
	if needed == 0 {
		return []string{}, nil
	}
	buf := make([]byte, needed)
	r, _, e := procEnumPrinters.Call(flags, 0, 4,
		uintptr(unsafe.Pointer(&buf[0])), uintptr(needed), //nolint:gosec
		uintptr(unsafe.Pointer(&needed)), uintptr(unsafe.Pointer(&returned))) //nolint:gosec
	if r == 0 {
		return nil, fmt.Errorf("list printers: %w", e)
	}
	out := make([]string, 0, returned)
	size := unsafe.Sizeof(printerInfo4{})
	for i := range uintptr(returned) {
		pi := (*printerInfo4)(unsafe.Pointer(&buf[i*size])) //nolint:gosec // safe: buf sized by EnumPrinters
		if pi.printerName != nil {
			out = append(out, utf16PtrToString(pi.printerName))
		}
	}
	return out, nil
}

func utf16PtrToString(p *uint16) string {
	n := 0
	for ptr := unsafe.Pointer(p); *(*uint16)(ptr) != 0; n++ { //nolint:gosec // walks a NUL-terminated Win32 string
		ptr = unsafe.Add(ptr, 2) //nolint:gosec // byte advance matches UTF-16 units
	}
	return syscall.UTF16ToString(unsafe.Slice(p, n)) //nolint:gosec // n counted from the same buffer
}
