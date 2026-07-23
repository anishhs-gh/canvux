// Package imgembed converts raster images (PNG/JPEG/GIF) into scene objects:
// a downsampled grid of filled rects, with horizontal same-color runs merged
// so typical images stay in the hundreds of objects, not thousands.
package imgembed

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"

	"github.com/anishhs-gh/canvux/internal/geom"
	"github.com/anishhs-gh/canvux/internal/scene"
)

// FromFile loads and converts an image. maxCols caps the world-space width;
// height follows the aspect ratio (halved, since world units are ~square but
// source pixels dominate horizontally in terminals).
func FromFile(path string, maxCols, layer int) ([]*scene.Object, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	return Convert(img, maxCols, layer), nil
}

// Convert downsamples img to at most maxCols columns and emits run-merged
// rect objects, one world unit per sampled pixel.
func Convert(img image.Image, maxCols, layer int) []*scene.Object {
	if maxCols < 4 {
		maxCols = 4
	}
	b := img.Bounds()
	sw, sh := b.Dx(), b.Dy()
	if sw == 0 || sh == 0 {
		return nil
	}
	cols := minInt(maxCols, sw)
	rows := maxInt(1, sh*cols/maxInt(1, sw))
	if rows > maxCols { // keep tall images bounded too
		rows = maxCols
		cols = maxInt(1, sw*rows/sh)
	}

	// Average-pool each cell, quantize slightly so runs merge well.
	grid := make([][]cell, rows)
	for cy := 0; cy < rows; cy++ {
		grid[cy] = make([]cell, cols)
		for cx := 0; cx < cols; cx++ {
			x0 := b.Min.X + cx*sw/cols
			x1 := b.Min.X + (cx+1)*sw/cols
			y0 := b.Min.Y + cy*sh/rows
			y1 := b.Min.Y + (cy+1)*sh/rows
			var rSum, gSum, bSum, aSum, n uint64
			for y := y0; y < maxInt(y0+1, y1); y++ {
				for x := x0; x < maxInt(x0+1, x1); x++ {
					r, g, bb, a := img.At(x, y).RGBA()
					rSum += uint64(r >> 8)
					gSum += uint64(g >> 8)
					bSum += uint64(bb >> 8)
					aSum += uint64(a >> 8)
					n++
				}
			}
			c := cell{
				color: scene.Color{
					R: quant(uint8(rSum / n)),
					G: quant(uint8(gSum / n)),
					B: quant(uint8(bSum / n)),
				},
				opaque: aSum/n > 32,
			}
			grid[cy][cx] = c
		}
	}

	// Merge horizontal runs of identical color into single rects.
	var objs []*scene.Object
	for cy := 0; cy < rows; cy++ {
		cx := 0
		for cx < cols {
			c := grid[cy][cx]
			if !c.opaque {
				cx++
				continue
			}
			run := cx + 1
			for run < cols && grid[cy][run].opaque && grid[cy][run].color == c.color {
				run++
			}
			objs = append(objs, &scene.Object{
				Kind:   scene.KindRect,
				P1:     geom.V(float64(cx), float64(cy)),
				P2:     geom.V(float64(run), float64(cy+1)),
				Stroke: c.color, Fill: c.color, Filled: true,
				StrokeWidth: 0.5, Opacity: 1, Layer: layer,
			})
			cx = run
		}
	}
	return objs
}

type cell struct {
	color  scene.Color
	opaque bool
}

// quant rounds channels to 16 levels so adjacent cells merge into runs.
func quant(v uint8) uint8 {
	q := (uint16(v) + 8) / 17 * 17
	if q > 255 {
		q = 255
	}
	return uint8(q)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
