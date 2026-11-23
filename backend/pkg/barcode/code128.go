package barcode

import "fmt"

// Code 128 symbol patterns: each entry is six module widths alternating
// bar/space/bar/space/bar/space for symbols 0–105. The stop symbol (106)
// uses the fixed 7-width pattern stopPattern.
var code128 = [...]string{
	"212222", "222122", "222221", "121223", "121322", "131222", "122213", "122312",
	"132212", "221213", "221312", "231212", "112232", "122132", "122231", "113222",
	"123122", "123221", "223211", "221132", "221231", "213212", "223112", "312131",
	"311222", "321122", "321221", "312212", "322112", "322211", "212123", "212321",
	"232121", "111323", "131123", "131321", "112313", "132113", "132311", "211313",
	"231113", "231311", "112133", "112331", "132131", "113123", "113321", "133121",
	"313121", "211331", "231131", "213113", "213311", "213131", "311123", "311321",
	"331121", "312113", "312311", "332111", "314111", "221411", "431411", "111224",
	"111422", "121124", "121421", "141122", "141221", "112214", "112412", "122114",
	"122411", "142112", "142211", "241211", "221114", "413111", "241112", "134111",
	"111242", "121142", "121241", "114212", "124112", "124211", "411212", "421112",
	"421211", "212141", "214121", "412121", "111143", "111341", "131141", "114113",
	"114311", "411113", "411311", "113141", "114131", "311141", "411131", "211412",
	"211214", "211232", "211233",
}

const (
	startB       = 104
	stopPattern  = "2331112"
	quietModules = 10 // Code 128 spec quiet zone per side
)

// Code128 encodes text as Code 128 set B and returns the full symbol pattern:
// a flat slice of booleans (true = bar/ink) including quiet zones. It returns
// an error for empty input or characters outside printable ASCII 32–126.
func Code128(text string) ([]bool, error) {
	if text == "" {
		return nil, fmt.Errorf("barcode: empty input")
	}
	if len(text) > 48 {
		return nil, fmt.Errorf("barcode: input too long (%d > 48)", len(text))
	}
	symbols := make([]int, 0, len(text)+3)
	symbols = append(symbols, startB)
	sum := startB
	for i := 0; i < len(text); i++ {
		v := int(text[i]) - 32
		if v < 0 || v > 94 {
			return nil, fmt.Errorf("barcode: character %q is not printable ASCII", text[i])
		}
		symbols = append(symbols, v)
		sum += (i + 1) * v // position weight starts at 1
	}
	symbols = append(symbols, sum%103)

	out := make([]bool, 0, quietModules*2+len(symbols)*11+13)
	out = append(out, make([]bool, quietModules)...)
	for _, s := range symbols {
		appendPattern(&out, code128[s])
	}
	appendPattern(&out, stopPattern)
	out = append(out, make([]bool, quietModules)...)
	return out, nil
}

// appendPattern expands a widths string ("212222") into bar=true/space=false
// modules starting with a bar.
func appendPattern(out *[]bool, widths string) {
	bar := true
	for i := 0; i < len(widths); i++ {
		for n := int(widths[i] - '0'); n > 0; n-- {
			*out = append(*out, bar)
		}
		bar = !bar
	}
}
