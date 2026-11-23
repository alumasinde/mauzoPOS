package barcode

import "testing"

func TestClassifyValid(t *testing.T) {
	cases := []struct {
		code  string
		typ   Type
		valid bool
	}{
		{"5901234123457", TypeEAN13, true},  // canonical valid EAN-13
		{"5901234123458", TypeEAN13, false}, // bad check digit
		{"96385074", TypeEAN8, true},        // valid EAN-8
		{"96385075", TypeEAN8, false},
		{"036000291452", TypeUPCA, true}, // valid UPC-A
		{"036000291453", TypeUPCA, false},
		{"ABC-123", TypeOther, true}, // opaque code, no checksum
		{"", TypeOther, false},
	}
	for _, c := range cases {
		if got := Classify(c.code); got != c.typ {
			t.Errorf("Classify(%q) = %q, want %q", c.code, got, c.typ)
		}
		if got := Valid(c.code); got != c.valid {
			t.Errorf("Valid(%q) = %v, want %v", c.code, got, c.valid)
		}
	}
}

func TestCheckDigit(t *testing.T) {
	if CheckDigit("590123412345") != 7 {
		t.Fatal("EAN-13 payload should yield check digit 7")
	}
	if CheckDigit("03600029145") != 2 {
		t.Fatal("UPC-A payload should yield check digit 2")
	}
}

func TestParseScale(t *testing.T) {
	cfg := ScaleConfig{Prefixes: []string{"21"}, Embedded: EmbeddedPrice}
	// 21 + item 12345 + price 15050 (KES 150.50) + check digit
	payload := "211234515050"
	code := payload + string(rune('0'+CheckDigit(payload)))
	s, ok := ParseScale(code, cfg)
	if !ok {
		t.Fatalf("ParseScale(%q) rejected a valid label", code)
	}
	if s.Item != "12345" || s.Value.String() != "150.5" {
		t.Fatalf("got %+v, want item 12345 value 150.5", s)
	}

	// Weight encoding: 21 + 00042 + 1250 g → 1.25 kg
	wcfg := ScaleConfig{Prefixes: []string{"21"}, Embedded: EmbeddedWeight}
	payload = "210004201250"
	code = payload + string(rune('0'+CheckDigit(payload)))
	s, ok = ParseScale(code, wcfg)
	if !ok || s.Value.String() != "1.25" {
		t.Fatalf("weight parse got %+v ok=%v", s, ok)
	}

	// Wrong prefix → not a scale label
	if _, ok = ParseScale("5901234123457", cfg); ok {
		t.Fatal("normal EAN-13 must not parse as a scale label")
	}
	// Bad check digit → reject
	if _, ok = ParseScale("2112345150509", cfg); ok {
		t.Fatal("label with bad check digit must be rejected")
	}
}

func TestCode128(t *testing.T) {
	pat, err := Code128("A")
	if err != nil {
		t.Fatal(err)
	}
	// quiet(10) + start(11) + 'A'(11) + checksum(11) + stop(13) + quiet(10)
	if want := 10 + 11*3 + 13 + 10; len(pat) != want {
		t.Fatalf("len = %d, want %d", len(pat), want)
	}
	// quiet zones are blank
	for i := 0; i < 10; i++ {
		if pat[i] || pat[len(pat)-1-i] {
			t.Fatal("quiet zone is not blank")
		}
	}
	// start B pattern is 211214
	wantStart := "211214"
	pos := 10
	for i := 0; i < 6; i++ {
		w := int(wantStart[i] - '0')
		bar := i%2 == 0
		for n := 0; n < w; n++ {
			if pat[pos] != bar {
				t.Fatalf("start pattern mismatch at module %d", pos)
			}
			pos++
		}
	}
	if _, err = Code128(""); err == nil {
		t.Fatal("empty input must fail")
	}
	if _, err = Code128("héllo"); err == nil {
		t.Fatal("non-ASCII input must fail")
	}
}
