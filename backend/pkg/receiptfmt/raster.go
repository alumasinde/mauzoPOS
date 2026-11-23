package receiptfmt

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"math"

	// Decoders for logo uploads.
	_ "image/gif"
	_ "image/jpeg"
)

// Raster is a 1-bit image ready for a thermal head: Width×Height pixels,
// row-major, MSB first, rows padded to whole bytes; 1 = black.
type Raster struct {
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Bits   []byte `json:"bits"`
}

// BytesPerRow is the padded row size.
func (r Raster) BytesPerRow() int { return (r.Width + 7) / 8 }

// Valid reports whether Bits matches the dimensions.
func (r Raster) Valid() bool {
	return r.Width > 0 && r.Height > 0 && len(r.Bits) == r.BytesPerRow()*r.Height
}

// Logo limits: wide enough for a 58 mm roll, short enough not to waste paper.
const (
	LogoMaxWidth  = 384
	LogoMaxHeight = 200
)

// ErrBadImage is returned for unreadable or empty images.
var ErrBadImage = errors.New("receiptfmt: unsupported or empty image")

// LogoFromImage decodes a PNG/JPEG/GIF, scales it to fit maxW×maxH (never
// up), flattens transparency onto white and dithers it to 1 bit
// (Floyd–Steinberg), which keeps photos and gradients legible on thermal
// paper.
func LogoFromImage(data []byte, maxW, maxH int) (Raster, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return Raster{}, ErrBadImage
	}
	b := img.Bounds()
	if b.Dx() == 0 || b.Dy() == 0 {
		return Raster{}, ErrBadImage
	}
	scale := math.Min(1, math.Min(float64(maxW)/float64(b.Dx()), float64(maxH)/float64(b.Dy())))
	w := max(1, int(math.Round(float64(b.Dx())*scale)))
	h := max(1, int(math.Round(float64(b.Dy())*scale)))

	// Area-average downscale into a luminance buffer (0 = black, 1 = white).
	lum := make([]float64, w*h)
	for y := range h {
		y0 := b.Min.Y + int(float64(y)/scale)
		y1 := max(y0+1, b.Min.Y+int(float64(y+1)/scale))
		for x := range w {
			x0 := b.Min.X + int(float64(x)/scale)
			x1 := max(x0+1, b.Min.X+int(float64(x+1)/scale))
			var sum float64
			n := 0
			for sy := y0; sy < y1 && sy < b.Max.Y; sy++ {
				for sx := x0; sx < x1 && sx < b.Max.X; sx++ {
					sum += luminance(img.At(sx, sy))
					n++
				}
			}
			if n > 0 {
				lum[y*w+x] = sum / float64(n)
			} else {
				lum[y*w+x] = 1
			}
		}
	}

	r := Raster{Width: w, Height: h}
	bpr := r.BytesPerRow()
	r.Bits = make([]byte, bpr*h)
	for y := range h {
		for x := range w {
			old := lum[y*w+x]
			black := old < 0.5
			nv := 1.0
			if black {
				nv = 0
				r.Bits[y*bpr+x/8] |= 0x80 >> (x % 8)
			}
			e := old - nv
			spread := func(dx, dy int, f float64) {
				nx, ny := x+dx, y+dy
				if nx >= 0 && nx < w && ny < h {
					lum[ny*w+nx] += e * f
				}
			}
			spread(1, 0, 7.0/16)
			spread(-1, 1, 3.0/16)
			spread(0, 1, 5.0/16)
			spread(1, 1, 1.0/16)
		}
	}
	return r, nil
}

// luminance returns 0 (black) … 1 (white), compositing alpha over white.
func luminance(c color.Color) float64 {
	r, g, b, a := c.RGBA()
	if a == 0 {
		return 1
	}
	af := float64(a) / 0xffff
	l := (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 0xffff // premultiplied
	return l + (1 - af)
}

// PNG renders the raster as a black-on-white PNG (designer preview).
func (r Raster) PNG() ([]byte, error) {
	if !r.Valid() {
		return nil, ErrBadImage
	}
	img := image.NewGray(image.Rect(0, 0, r.Width, r.Height))
	bpr := r.BytesPerRow()
	for y := range r.Height {
		for x := range r.Width {
			v := uint8(255)
			if r.Bits[y*bpr+x/8]&(0x80>>(x%8)) != 0 {
				v = 0
			}
			img.SetGray(x, y, color.Gray{Y: v})
		}
	}
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	return buf.Bytes(), err
}
