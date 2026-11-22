package money

import (
	"encoding/json"
	"testing"

	"github.com/shopspring/decimal"
)

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func TestRoundingModes(t *testing.T) {
	cases := []struct {
		in   string
		mode Mode
		want string
	}{
		{"0.125", HalfUp, "0.13"},
		{"-0.125", HalfUp, "-0.13"},
		{"0.125", HalfEven, "0.12"},
		{"0.135", HalfEven, "0.14"},
		{"0.121", Up, "0.13"},
		{"-0.121", Up, "-0.13"},
		{"0.129", Down, "0.12"},
		{"-0.129", Down, "-0.12"},
	}
	for _, c := range cases {
		if got := MustParse(c.in).Round(c.mode).String(); got != c.want {
			t.Errorf("Round(%s, %s) = %s, want %s", c.in, c.mode, got, c.want)
		}
	}
}

func TestNoFloatDrift(t *testing.T) {
	// 0.1 + 0.2 is the classic float failure.
	if got := MustParse("0.1").Add(MustParse("0.2")).String(); got != "0.30" {
		t.Fatalf("got %s", got)
	}
}

func TestWeightPricing(t *testing.T) {
	// 0.735 kg of beef at KSh 650/kg = 477.75
	got := MustParse("650").Mul(d("0.735")).Round(HalfUp)
	if got.String() != "477.75" {
		t.Fatalf("got %s", got)
	}
}

func TestInclusiveVATExtraction(t *testing.T) {
	// Price 750.34 incl. 16% VAT → net = 750.34 / 1.16, tax = gross - net
	gross := MustParse("750.34")
	net := gross.Div(d("1.16")).Round(HalfUp)
	tax := gross.Sub(net)
	if net.String() != "646.84" || tax.String() != "103.50" {
		t.Fatalf("net=%s tax=%s", net, tax)
	}
	if !net.Add(tax).Equal(gross) {
		t.Fatal("net + tax must equal gross")
	}
}

func TestRoundToIncrement(t *testing.T) {
	cases := []struct {
		in, inc string
		mode    Mode
		want    string
	}{
		{"750.34", "0.05", HalfUp, "750.35"},
		{"750.32", "0.05", HalfUp, "750.30"},
		{"750.34", "1", Down, "750.00"},
		{"750.01", "1", Up, "751.00"},
	}
	for _, c := range cases {
		got := MustParse(c.in).RoundToIncrement(MustParse(c.inc), c.mode).String()
		if got != c.want {
			t.Errorf("RoundToIncrement(%s, %s, %s) = %s, want %s", c.in, c.inc, c.mode, got, c.want)
		}
	}
}

func TestAllocateSumsExactly(t *testing.T) {
	parts, err := Allocate(MustParse("100.00"), []decimal.Decimal{d("1"), d("1"), d("1")})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"33.34", "33.33", "33.33"}
	for i, p := range parts {
		if p.String() != want[i] {
			t.Fatalf("parts = %v", parts)
		}
	}
	if !Sum(parts...).Equal(MustParse("100.00")) {
		t.Fatal("parts do not sum to total")
	}
}

func TestAllocateProportionalAndNegative(t *testing.T) {
	// Spread a -10.00 discount across lines worth 110, 50, 320.
	parts, err := Allocate(MustParse("-10"), []decimal.Decimal{d("110"), d("50"), d("320")})
	if err != nil {
		t.Fatal(err)
	}
	if !Sum(parts...).Equal(MustParse("-10")) {
		t.Fatalf("sum = %s", Sum(parts...))
	}
	if parts[2].String() != "-6.67" {
		t.Fatalf("largest line share = %s", parts[2])
	}
}

func TestAllocateRejectsBadWeights(t *testing.T) {
	if _, err := Allocate(MustParse("1"), nil); err == nil {
		t.Fatal("expected error for no weights")
	}
	if _, err := Allocate(MustParse("1"), []decimal.Decimal{d("0")}); err == nil {
		t.Fatal("expected error for zero weights")
	}
	if _, err := Allocate(MustParse("1"), []decimal.Decimal{d("-1"), d("2")}); err == nil {
		t.Fatal("expected error for negative weight")
	}
}

func TestJSONIsString(t *testing.T) {
	b, _ := json.Marshal(struct{ Total Money }{MustParse("750.34")})
	if string(b) != `{"Total":"750.34"}` {
		t.Fatalf("got %s", b)
	}
	var v struct{ Total Money }
	if err := json.Unmarshal([]byte(`{"Total":12.5}`), &v); err != nil || v.Total.String() != "12.50" {
		t.Fatalf("unmarshal: %v %s", err, v.Total)
	}
}

func TestScanFromDriverBytes(t *testing.T) {
	var m Money
	if err := m.Scan([]byte("320.00")); err != nil || m.String() != "320.00" {
		t.Fatalf("scan: %v %s", err, m)
	}
}

func TestFormat(t *testing.T) {
	if got := MustParse("1234567.5").Format("KSh"); got != "KSh 1,234,567.50" {
		t.Fatalf("got %q", got)
	}
	if got := MustParse("-90").Format(""); got != "-90.00" {
		t.Fatalf("got %q", got)
	}
}

func TestCents(t *testing.T) {
	if FromCents(75034).String() != "750.34" || MustParse("750.345").Cents() != 75035 {
		t.Fatal("cents conversion wrong")
	}
}
