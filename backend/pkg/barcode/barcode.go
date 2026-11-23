// Package barcode validates and decodes retail barcodes. It is pure Go with
// no app dependencies so the API and the offline POS share the exact same
// rules: EAN-8/EAN-13/UPC-A check-digit validation, weighted (scale) label
// decoding, and Code 128 pattern generation for printed shelf labels.
package barcode

// Type identifies a barcode symbology.
type Type string

// Symbologies we can validate by check digit, plus TypeOther for opaque codes.
const (
	TypeEAN8  Type = "ean8"
	TypeEAN13 Type = "ean13"
	TypeUPCA  Type = "upca"
	TypeOther Type = "other" // Code39/128, QR — no checksum we can verify
)

// MaxLen is the longest barcode we store.
const MaxLen = 64

// Classify reports the symbology of code by length and charset. It does not
// verify the check digit — use Valid for that.
func Classify(code string) Type {
	if !isDigits(code) {
		return TypeOther
	}
	switch len(code) {
	case 8:
		return TypeEAN8
	case 12:
		return TypeUPCA
	case 13:
		return TypeEAN13
	default:
		return TypeOther
	}
}

// Valid reports whether code is a well-formed barcode. For EAN/UPC the check
// digit must match; other symbologies pass when non-empty and within MaxLen.
func Valid(code string) bool {
	switch Classify(code) {
	case TypeEAN8, TypeEAN13, TypeUPCA:
		return CheckDigit(code[:len(code)-1]) == digit(code[len(code)-1])
	default:
		return len(code) > 0 && len(code) <= MaxLen
	}
}

// CheckDigit computes the GS1 modulo-10 check digit for digits (the payload
// without the check digit). Works for EAN-8, EAN-13 and UPC-A.
func CheckDigit(digits string) int {
	sum := 0
	// From the rightmost payload digit the weights alternate 3,1,3,1… for all
	// GS1 symbologies regardless of total length.
	for i, weight := 0, 3; i < len(digits); i++ {
		sum += digit(digits[len(digits)-1-i]) * weight
		if weight == 3 {
			weight = 1
		} else {
			weight = 3
		}
	}
	return (10 - sum%10) % 10
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func digit(b byte) int { return int(b - '0') }
