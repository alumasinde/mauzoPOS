package receiptfmt

import (
	"bytes"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

func TestBuiltinsRenderWithinWidth(t *testing.T) {
	for _, b := range Builtins {
		out := Render(b.Template, SampleDoc())
		if out.Width != b.Template.Width() {
			t.Fatalf("%s: width %d", b.Key, out.Width)
		}
		for i, l := range out.Lines {
			if l.Kind != LineText {
				continue
			}
			limit := out.Width
			if l.Size == "wide" || l.Size == "double" {
				limit = out.Width / 2
			}
			if n := runeLen(l.Text); n > limit {
				t.Errorf("%s line %d is %d chars (limit %d): %q", b.Key, i, n, limit, l.Text)
			}
		}
		txt := out.Text()
		for _, want := range []string{"SAMPLE - NOT A RECEIPT", "1047", "Kabras Sugar 2kg", "1,250.00", "M-Pesa"} {
			if !strings.Contains(txt, want) {
				t.Errorf("%s: missing %q\n%s", b.Key, want, txt)
			}
		}
	}
}

func TestConditionsAndVariables(t *testing.T) {
	tpl := MustParse([]byte(`{"paper":58,"blocks":[
		{"type":"text","text":"PIN {{business.kra_pin}}","if":"business.kra_pin"},
		{"type":"text","text":"CUSTOMER {{sale.customer}}","if":"has.customer"},
		{"type":"text","text":"WALK-IN","if":"!has.customer"},
		{"type":"text","text":"TILL {{extra.mpesa_till}}","if":"extra.mpesa_till"}
	]}`))
	d := SampleDoc()
	d.Flags.Sample = false
	d.Business.KRAPin = ""
	d.Sale.Customer = ""
	txt := Render(tpl, d).Text()
	if strings.Contains(txt, "PIN") || strings.Contains(txt, "CUSTOMER") {
		t.Fatalf("hidden blocks rendered:\n%s", txt)
	}
	if !strings.Contains(txt, "WALK-IN") || !strings.Contains(txt, "TILL 123456") {
		t.Fatalf("expected blocks missing:\n%s", txt)
	}
}

func TestBannersCannotBeRemoved(t *testing.T) {
	tpl := MustParse([]byte(`{"paper":80,"blocks":[{"type":"text","text":"hello"}]}`))
	d := SampleDoc()
	d.Flags = Flags{Reprint: true, CopyNo: 2, Offline: true, Voided: true}
	txt := Render(tpl, d).Text()
	for _, want := range []string{"*** REPRINT 2 ***", "PROVISIONAL - OFFLINE", "*** VOIDED ***"} {
		if strings.Count(txt, want) != 2 {
			t.Errorf("want %q top and bottom:\n%s", want, txt)
		}
	}
	marked := Mark(Render(tpl, SampleDoc()), ReprintBanner(1))
	if marked.Lines[0].Text == "" || !strings.Contains(marked.Lines[0].Text, "REPRINT 1") {
		t.Fatalf("Mark: %+v", marked.Lines[0])
	}
}

func TestValidate(t *testing.T) {
	cases := map[string]string{
		`{"paper":70,"blocks":[{"type":"divider"}]}`:                            "paper must be 58 or 80",
		`{"paper":80,"blocks":[]}`:                                              "at least one block",
		`{"paper":80,"blocks":[{"type":"bogus"}]}`:                              "unknown block type",
		`{"paper":80,"blocks":[{"type":"text","text":"{{nope.x}}"}]}`:           "unknown variable",
		`{"paper":80,"blocks":[{"type":"text","text":"x","if":"has.unicorn"}]}`: "unknown condition",
		`{"paper":80,"blocks":[{"type":"feed","lines":50}]}`:                    "lines must be",
		`{"paper":80,"columns":10,"blocks":[{"type":"divider"}]}`:               "columns must be",
	}
	for body, want := range cases {
		_, err := Parse([]byte(body))
		var ve *ValidationError
		if !errors.As(err, &ve) || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: want %q, got %v", body, want, err)
		}
	}
	if _, err := Parse([]byte(`{"version":99,"paper":80,"blocks":[{"type":"divider"}]}`)); err == nil {
		t.Error("future schema version accepted")
	}
	if _, err := Parse([]byte(`{"paper":80,"blocks":[{"type":"divider","colour":"red"}]}`)); err == nil {
		t.Error("unknown field accepted")
	}
	// Any extra.* key is allowed.
	if _, err := Parse([]byte(`{"paper":80,"blocks":[{"type":"text","text":"{{extra.loyalty_note}}"}]}`)); err != nil {
		t.Errorf("extra var: %v", err)
	}
}

func TestLayoutHelpers(t *testing.T) {
	if got := Money("1234567.5"); got != "1,234,567.50" {
		t.Errorf("Money: %q", got)
	}
	if got := Money("-40"); got != "-40.00" {
		t.Errorf("Money neg: %q", got)
	}
	if got := Qty("2.5000"); got != "2.5" {
		t.Errorf("Qty: %q", got)
	}
	lines := pair("Very long product name that wraps", "1,000.00", 20)
	if len(lines) < 2 || !strings.HasSuffix(lines[len(lines)-1], "1,000.00") {
		t.Errorf("pair: %q", lines)
	}
	for _, l := range lines {
		if runeLen(l) > 20 {
			t.Errorf("pair overflow: %q", l)
		}
	}
	if w := wrap("abcdefghijklmnopqrstuvwxyz", 10); len(w) != 3 || w[0] != "abcdefghij" {
		t.Errorf("wrap long word: %q", w)
	}
}

func TestDividersCollapse(t *testing.T) {
	tpl := MustParse([]byte(`{"paper":58,"blocks":[
		{"type":"divider"},{"type":"fiscal"},{"type":"divider","char":"="},{"type":"text","text":"end"}]}`))
	d := SampleDoc()
	d.Flags.Sample = false
	out := Render(tpl, d)
	rules := 0
	for _, l := range out.Lines {
		if l.Kind == LineText && isRule(l.Text) {
			rules++
		}
	}
	if rules != 1 {
		t.Fatalf("want 1 rule, got %d:\n%s", rules, out.Text())
	}
}

func TestTaxBreakdown(t *testing.T) {
	got := TaxBreakdown(SampleDoc().Items, true)
	if len(got) != 2 || got[0].Rate != "16" || got[0].Tax != "115.87" || got[0].Taxable != "724.13" {
		t.Fatalf("breakdown: %+v", got)
	}
	if got[1].Rate != "0" || got[1].Taxable != "410.00" {
		t.Fatalf("zero-rated: %+v", got[1])
	}
}

func TestEncodeESCPOS(t *testing.T) {
	b, _ := BuiltinByKey("supermarket")
	d := SampleDoc()
	d.Flags.HasLogo = true
	logo := &Raster{Width: 16, Height: 2, Bits: []byte{0xFF, 0x00, 0x0F, 0xF0}}
	out := EncodeESCPOS(Render(b.Template, d), PrintOptions{Logo: logo, Cut: true, OpenDrawer: true})

	mustHave := map[string][]byte{
		"init":         {0x1B, '@'},
		"drawer kick":  {0x1B, 'p', 0, 25, 250},
		"center":       {0x1B, 'a', 1},
		"double size":  {0x1D, '!', 0x11},
		"bold on":      {0x1B, 'E', 1},
		"raster logo":  {0x1D, 'v', '0', 0, 2, 0, 2, 0, 0xFF, 0x00, 0x0F, 0xF0},
		"code128":      append([]byte{0x1D, 'k', 73, 6}, []byte("{B1047")...),
		"partial cut":  {0x1D, 'V', 66, 0},
		"receipt text": []byte("Kabras Sugar 2kg"),
	}
	for name, seq := range mustHave {
		if !bytes.Contains(out, seq) {
			t.Errorf("missing %s % X", name, seq)
		}
	}
	if !bytes.HasSuffix(out, []byte{0x1D, 'V', 66, 0}) {
		t.Error("job must end with the cut")
	}
	// Text is always 7-bit (no mojibake). Drawer-kick timings and raster
	// bits are binary, so check a job without them.
	plain := SampleDoc()
	plain.Items[0].Name = "Café – Jambo"
	for _, c := range EncodeESCPOS(Render(b.Template, plain), PrintOptions{Cut: true}) {
		if c >= 0x80 {
			t.Fatalf("non-ASCII byte 0x%X in printer stream", c)
		}
	}
}

func TestQRCommand(t *testing.T) {
	tpl := MustParse([]byte(`{"paper":80,"blocks":[{"type":"qr","data":"https://x.ke/{{sale.receipt_no}}","qr_size":5}]}`))
	out := EncodeESCPOS(Render(tpl, SampleDoc()), PrintOptions{})
	data := "https://x.ke/1047"
	store := append([]byte{0x1D, '(', 'k', byte(len(data) + 3), 0, 49, 80, 48}, data...)
	for name, seq := range map[string][]byte{
		"model": {0x1D, '(', 'k', 4, 0, 49, 65, 50, 0},
		"size":  {0x1D, '(', 'k', 3, 0, 49, 67, 5},
		"store": store,
		"print": {0x1D, '(', 'k', 3, 0, 49, 81, 48},
	} {
		if !bytes.Contains(out, seq) {
			t.Errorf("QR %s missing % X", name, seq)
		}
	}
}

func TestASCII(t *testing.T) {
	if got := ASCII("Café – “Jambo” × 2 · ok"); got != `Cafe - "Jambo" x 2 - ok` {
		t.Errorf("ASCII: %q", got)
	}
	if got := ASCII("中"); got != "?" {
		t.Errorf("ASCII fallback: %q", got)
	}
}

func TestLogoFromImage(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 800, 100))
	for y := range 100 {
		for x := range 800 {
			c := color.NRGBA{255, 255, 255, 255}
			if x < 400 {
				c = color.NRGBA{0, 0, 0, 255}
			}
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	r, err := LogoFromImage(buf.Bytes(), LogoMaxWidth, LogoMaxHeight)
	if err != nil {
		t.Fatal(err)
	}
	if r.Width != 384 || r.Height != 48 || !r.Valid() {
		t.Fatalf("size %dx%d valid=%v", r.Width, r.Height, r.Valid())
	}
	// Left half black, right half white.
	if r.Bits[0] != 0xFF || r.Bits[r.BytesPerRow()-1] != 0x00 {
		t.Fatalf("pixels: %X … %X", r.Bits[0], r.Bits[r.BytesPerRow()-1])
	}
	if _, err := r.PNG(); err != nil {
		t.Fatal(err)
	}
	if _, err := LogoFromImage([]byte("not an image"), 10, 10); !errors.Is(err, ErrBadImage) {
		t.Fatalf("bad image: %v", err)
	}
}

func TestKitFallsBackToBuiltin(t *testing.T) {
	k := Kit{Template: json.RawMessage(`{"broken":`), BuiltinKey: "minimart"}
	if got := k.ParsedTemplate(); got.Paper != 58 {
		t.Fatalf("fallback paper %d", got.Paper)
	}
	k = Kit{Logo: &Raster{Width: 8, Height: 1, Bits: []byte{1}}}
	if !k.Base().Flags.HasLogo {
		t.Fatal("HasLogo not set from kit")
	}
}
