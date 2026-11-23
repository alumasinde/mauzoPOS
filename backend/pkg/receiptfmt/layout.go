package receiptfmt

import (
	"strings"
	"unicode/utf8"
)

// Money formats a decimal string as "1,234.50" ("" stays "", junk → "0.00").
func Money(s string) string {
	if strings.TrimSpace(s) == "" {
		return ""
	}
	d := dec(s).Round(2)
	neg := d.IsNegative()
	whole := d.Abs().Truncate(0).String()
	frac := d.Abs().Sub(d.Abs().Truncate(0)).StringFixed(2)[1:] // ".50"
	var b strings.Builder
	for i, c := range whole {
		if i > 0 && (len(whole)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	out := b.String() + frac
	if neg {
		out = "-" + out
	}
	return out
}

// Qty formats a quantity without trailing zeros: "2", "1.5", "0.125".
func Qty(s string) string {
	if strings.TrimSpace(s) == "" {
		return ""
	}
	return dec(s).Round(3).String()
}

func runeLen(s string) int { return utf8.RuneCountInString(s) }

func padRight(s string, w int) string {
	s = truncate(s, w)
	return s + strings.Repeat(" ", w-runeLen(s))
}

func padLeft(s string, w int) string {
	s = truncate(s, w)
	return strings.Repeat(" ", w-runeLen(s)) + s
}

func center(s string, w int) string {
	s = truncate(s, w)
	left := (w - runeLen(s)) / 2
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", w-runeLen(s)-left)
}

func align(s string, w int, a string) string {
	switch a {
	case "center":
		return center(s, w)
	case "right":
		return padLeft(s, w)
	}
	return padRight(s, w)
}

func truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if runeLen(s) <= w {
		return s
	}
	r := []rune(s)
	return string(r[:w])
}

// wrap breaks s into lines of at most w runes, on spaces where possible.
func wrap(s string, w int) []string {
	s = strings.TrimRight(s, " ")
	if s == "" {
		return []string{""}
	}
	var out []string
	var cur []rune
	for _, word := range strings.Fields(s) {
		wr := []rune(word)
		for len(wr) > w { // a single word longer than the line
			if len(cur) > 0 {
				out = append(out, string(cur))
				cur = nil
			}
			out = append(out, string(wr[:w]))
			wr = wr[w:]
		}
		switch {
		case len(cur) == 0:
			cur = wr
		case len(cur)+1+len(wr) <= w:
			cur = append(append(cur, ' '), wr...)
		default:
			out = append(out, string(cur))
			cur = wr
		}
	}
	if len(cur) > 0 {
		out = append(out, string(cur))
	}
	// Preserve deliberate leading indentation ("  2 x 100.00").
	if lead := len(s) - len(strings.TrimLeft(s, " ")); lead > 0 && len(out) > 0 && lead+runeLen(out[0]) <= w {
		out[0] = strings.Repeat(" ", lead) + out[0]
	}
	return out
}

// pairOne puts label left and value right on one line (label truncated).
func pairOne(label, value string, w int) string {
	gap := w - runeLen(value) - 1
	if gap < 1 {
		return padLeft(value, w)
	}
	return padRight(label, gap) + " " + value
}

// pair lays out label … value, wrapping a long label and keeping the value
// right-aligned on the label's last line (or its own line when no room).
func pair(label, value string, w int) []string {
	if runeLen(label)+1+runeLen(value) <= w {
		return []string{pairOne(label, value, w)}
	}
	lines := wrap(label, w)
	last := lines[len(lines)-1]
	if runeLen(last)+1+runeLen(value) <= w {
		lines[len(lines)-1] = pairOne(last, value, w)
		return lines
	}
	for _, v := range wrap(value, w) {
		lines = append(lines, padLeft(v, w))
	}
	return lines
}
