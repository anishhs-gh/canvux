// Package export renders documents to raster images (PNG) by reusing the
// scene rasterizer against an image-backed surface.
package export

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"

	"github.com/anishhs-gh/canvux/internal/render"
	"github.com/anishhs-gh/canvux/internal/scene"
)

// imageSurface adapts an *image.RGBA to the render.Surface interface.
type imageSurface struct{ img *image.RGBA }

func (s imageSurface) Size() (int, int) {
	b := s.img.Bounds()
	return b.Dx(), b.Dy()
}

func (s imageSurface) Set(x, y int, c scene.Color, a float64) {
	if !(image.Point{x, y}.In(s.img.Bounds())) || a <= 0 {
		return
	}
	old := s.img.RGBAAt(x, y)
	inv := 1 - a
	s.img.SetRGBA(x, y, color.RGBA{
		R: uint8(float64(c.R)*a + float64(old.R)*inv),
		G: uint8(float64(c.G)*a + float64(old.G)*inv),
		B: uint8(float64(c.B)*a + float64(old.B)*inv),
		A: 255,
	})
}

// PNG renders the document to w at `scale` pixels per world unit against
// background bg.
func PNG(w io.Writer, d *scene.Doc, scale float64, bg scene.Color) error {
	bounds := d.ContentBounds().Expand(4)
	if bounds.W() <= 0 || bounds.H() <= 0 {
		return fmt.Errorf("document is empty, nothing to export")
	}
	pw := int(math.Ceil(bounds.W() * scale))
	ph := int(math.Ceil(bounds.H() * scale))
	const maxDim = 16384
	if pw > maxDim || ph > maxDim {
		return fmt.Errorf("output would be %dx%d px; lower --scale (max %d px per side)", pw, ph, maxDim)
	}
	img := image.NewRGBA(image.Rect(0, 0, pw, ph))
	surf := imageSurface{img}
	for y := 0; y < ph; y++ {
		for x := 0; x < pw; x++ {
			img.SetRGBA(x, y, color.RGBA{bg.R, bg.G, bg.B, 255})
		}
	}
	view := render.View{Center: bounds.Center(), Zoom: scale, W: pw, H: ph}
	for _, o := range d.VisibleObjects() {
		render.DrawObject(surf, view, o)
		if o.Kind == scene.KindText {
			drawTextPx(surf, view, o)
		}
	}
	return png.Encode(w, img)
}

// drawTextPx renders text objects as simple 3x5 bitmap glyphs so PNG exports
// are not missing labels entirely.
func drawTextPx(s render.Surface, v render.View, o *scene.Object) {
	origin := v.ToPixel(o.P1)
	cellW := v.Zoom // one world unit per character cell
	scale := math.Max(1, cellW/4)
	x := origin.X
	for _, r := range o.Text {
		g, ok := microFont[r]
		if !ok {
			g = microFont['?']
		}
		for row := 0; row < 5; row++ {
			for col := 0; col < 3; col++ {
				if g[row]&(1<<(2-col)) == 0 {
					continue
				}
				for dy := 0; dy < int(scale); dy++ {
					for dx := 0; dx < int(scale); dx++ {
						s.Set(int(x)+col*int(scale)+dx, int(origin.Y)+row*int(scale)+dy, o.Stroke, o.Opacity)
					}
				}
			}
		}
		x += cellW
	}
}

// microFont is a tiny 3x5 uppercase bitmap font (rows of 3 bits, MSB left).
var microFont = map[rune][5]uint8{
	'A': {0b010, 0b101, 0b111, 0b101, 0b101}, 'B': {0b110, 0b101, 0b110, 0b101, 0b110},
	'C': {0b011, 0b100, 0b100, 0b100, 0b011}, 'D': {0b110, 0b101, 0b101, 0b101, 0b110},
	'E': {0b111, 0b100, 0b110, 0b100, 0b111}, 'F': {0b111, 0b100, 0b110, 0b100, 0b100},
	'G': {0b011, 0b100, 0b101, 0b101, 0b011}, 'H': {0b101, 0b101, 0b111, 0b101, 0b101},
	'I': {0b111, 0b010, 0b010, 0b010, 0b111}, 'J': {0b001, 0b001, 0b001, 0b101, 0b010},
	'K': {0b101, 0b110, 0b100, 0b110, 0b101}, 'L': {0b100, 0b100, 0b100, 0b100, 0b111},
	'M': {0b101, 0b111, 0b111, 0b101, 0b101}, 'N': {0b101, 0b111, 0b111, 0b111, 0b101},
	'O': {0b010, 0b101, 0b101, 0b101, 0b010}, 'P': {0b110, 0b101, 0b110, 0b100, 0b100},
	'Q': {0b010, 0b101, 0b101, 0b011, 0b001}, 'R': {0b110, 0b101, 0b110, 0b110, 0b101},
	'S': {0b011, 0b100, 0b010, 0b001, 0b110}, 'T': {0b111, 0b010, 0b010, 0b010, 0b010},
	'U': {0b101, 0b101, 0b101, 0b101, 0b111}, 'V': {0b101, 0b101, 0b101, 0b101, 0b010},
	'W': {0b101, 0b101, 0b111, 0b111, 0b101}, 'X': {0b101, 0b101, 0b010, 0b101, 0b101},
	'Y': {0b101, 0b101, 0b010, 0b010, 0b010}, 'Z': {0b111, 0b001, 0b010, 0b100, 0b111},
	'0': {0b010, 0b101, 0b101, 0b101, 0b010}, '1': {0b010, 0b110, 0b010, 0b010, 0b111},
	'2': {0b110, 0b001, 0b010, 0b100, 0b111}, '3': {0b110, 0b001, 0b010, 0b001, 0b110},
	'4': {0b101, 0b101, 0b111, 0b001, 0b001}, '5': {0b111, 0b100, 0b110, 0b001, 0b110},
	'6': {0b011, 0b100, 0b110, 0b101, 0b010}, '7': {0b111, 0b001, 0b010, 0b010, 0b010},
	'8': {0b010, 0b101, 0b010, 0b101, 0b010}, '9': {0b010, 0b101, 0b011, 0b001, 0b110},
	' ': {0, 0, 0, 0, 0}, '.': {0, 0, 0, 0, 0b010}, ',': {0, 0, 0, 0b010, 0b100},
	'-': {0, 0, 0b111, 0, 0}, '+': {0, 0b010, 0b111, 0b010, 0}, '!': {0b010, 0b010, 0b010, 0, 0b010},
	'?': {0b110, 0b001, 0b010, 0, 0b010}, ':': {0, 0b010, 0, 0b010, 0}, '/': {0b001, 0b001, 0b010, 0b100, 0b100},
	'(': {0b001, 0b010, 0b010, 0b010, 0b001}, ')': {0b100, 0b010, 0b010, 0b010, 0b100},
	'\'': {0b010, 0b010, 0, 0, 0}, '"': {0b101, 0b101, 0, 0, 0}, '=': {0, 0b111, 0, 0b111, 0},
	'_': {0, 0, 0, 0, 0b111}, '*': {0b101, 0b010, 0b101, 0, 0}, '#': {0b101, 0b111, 0b101, 0b111, 0b101},
}

func init() {
	// Lowercase maps to uppercase glyphs.
	for r := 'a'; r <= 'z'; r++ {
		if g, ok := microFont[r-32]; ok {
			microFont[r] = g
		}
	}
}
