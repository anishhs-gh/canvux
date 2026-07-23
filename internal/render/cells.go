package render

import (
	"fmt"
	"strings"

	"github.com/anishhs-gh/canvux/internal/scene"
)

// Mode selects how pixels are composited into terminal cells.
type Mode int

const (
	// ModeHalfBlock uses '▀' with fg/bg colors: 1x2 pixels per cell, full color.
	ModeHalfBlock Mode = iota
	// ModeBraille uses U+2800..28FF: 2x4 pixels per cell, one color per cell.
	ModeBraille
)

// PixelScale returns pixels-per-cell in x and y for the mode. Both modes give
// square pixels on a typical 1:2 terminal cell.
func (m Mode) PixelScale() (int, int) {
	if m == ModeBraille {
		return 2, 4
	}
	return 1, 2
}

func (m Mode) String() string {
	if m == ModeBraille {
		return "braille"
	}
	return "block"
}

// PixelBuf is an alpha-blending pixel Surface.
type PixelBuf struct {
	W, H int
	pix  []pixel
}

type pixel struct {
	r, g, b float64 // premultiplied by a
	a       float64
}

func NewPixelBuf(w, h int) *PixelBuf {
	return &PixelBuf{W: w, H: h, pix: make([]pixel, w*h)}
}

func (p *PixelBuf) Size() (int, int) { return p.W, p.H }

// Set blends c at opacity a over the existing pixel ("over" operator).
func (p *PixelBuf) Set(x, y int, c scene.Color, a float64) {
	if x < 0 || y < 0 || x >= p.W || y >= p.H || a <= 0 {
		return
	}
	i := y*p.W + x
	q := &p.pix[i]
	inv := 1 - a
	q.r = float64(c.R)*a + q.r*inv
	q.g = float64(c.G)*a + q.g*inv
	q.b = float64(c.B)*a + q.b*inv
	q.a = a + q.a*inv
}

// resolve returns the pixel blended over background bg, plus its coverage.
func (p *PixelBuf) resolve(x, y int, bg scene.Color) (scene.Color, float64) {
	q := p.pix[y*p.W+x]
	inv := 1 - q.a
	return scene.Color{
		R: uint8(q.r + float64(bg.R)*inv),
		G: uint8(q.g + float64(bg.G)*inv),
		B: uint8(q.b + float64(bg.B)*inv),
	}, q.a
}

// Cell is one terminal cell with explicit truecolor fg/bg.
type Cell struct {
	Ch     rune
	Fg, Bg scene.Color
}

// CellGrid is the final terminal frame: a W x H grid of styled cells.
type CellGrid struct {
	W, H  int
	Cells []Cell
}

func NewCellGrid(w, h int, bg scene.Color) *CellGrid {
	g := &CellGrid{W: w, H: h, Cells: make([]Cell, w*h)}
	for i := range g.Cells {
		g.Cells[i] = Cell{Ch: ' ', Fg: bg, Bg: bg}
	}
	return g
}

func (g *CellGrid) Set(x, y int, c Cell) {
	if x < 0 || y < 0 || x >= g.W || y >= g.H {
		return
	}
	g.Cells[y*g.W+x] = c
}

func (g *CellGrid) Get(x, y int) Cell {
	if x < 0 || y < 0 || x >= g.W || y >= g.H {
		return Cell{Ch: ' '}
	}
	return g.Cells[y*g.W+x]
}

// SetString writes s starting at (x, y) with the given colors.
func (g *CellGrid) SetString(x, y int, s string, fg, bg scene.Color) {
	for _, r := range s {
		g.Set(x, y, Cell{Ch: r, Fg: fg, Bg: bg})
		x++
	}
}

// FillRow paints an entire row with bg.
func (g *CellGrid) FillRow(y int, bg scene.Color) {
	for x := 0; x < g.W; x++ {
		g.Set(x, y, Cell{Ch: ' ', Fg: bg, Bg: bg})
	}
}

// FillRect paints a cell-space rectangle with bg.
func (g *CellGrid) FillRect(x0, y0, w, h int, bg scene.Color) {
	for y := y0; y < y0+h; y++ {
		for x := x0; x < x0+w; x++ {
			g.Set(x, y, Cell{Ch: ' ', Fg: bg, Bg: bg})
		}
	}
}

// Composite renders the pixel buffer into cell rows [rowOff, rowOff+rows).
func (g *CellGrid) Composite(p *PixelBuf, mode Mode, rowOff int, bg scene.Color) {
	sx, sy := mode.PixelScale()
	rows := p.H / sy
	for cy := 0; cy < rows; cy++ {
		for cx := 0; cx < p.W/sx; cx++ {
			if mode == ModeHalfBlock {
				top, ta := p.resolve(cx, cy*2, bg)
				bot, ba := p.resolve(cx, cy*2+1, bg)
				if ta == 0 && ba == 0 {
					continue // keep whatever is under (grid dots drawn as pixels too, so bg cell stays)
				}
				g.Set(cx, cy+rowOff, Cell{Ch: '▀', Fg: top, Bg: bot})
			} else {
				bits := 0
				var rSum, gSum, bSum, n float64
				for dy := 0; dy < 4; dy++ {
					for dx := 0; dx < 2; dx++ {
						c, a := p.resolve(cx*2+dx, cy*4+dy, bg)
						if a > 0.25 {
							bits |= brailleBit(dx, dy)
							rSum += float64(c.R)
							gSum += float64(c.G)
							bSum += float64(c.B)
							n++
						}
					}
				}
				if bits == 0 {
					continue
				}
				fg := scene.Color{R: uint8(rSum / n), G: uint8(gSum / n), B: uint8(bSum / n)}
				g.Set(cx, cy+rowOff, Cell{Ch: rune(0x2800 + bits), Fg: fg, Bg: bg})
			}
		}
	}
}

func brailleBit(dx, dy int) int {
	// Unicode braille dot numbering.
	table := [4][2]int{{0x01, 0x08}, {0x02, 0x10}, {0x04, 0x20}, {0x40, 0x80}}
	return table[dy][dx]
}

// ANSI serializes the grid into a terminal frame string.
func (g *CellGrid) ANSI() string {
	var b strings.Builder
	b.Grow(g.W * g.H * 8)
	var curFg, curBg scene.Color
	haveStyle := false
	for y := 0; y < g.H; y++ {
		if y > 0 {
			b.WriteString("\r\n")
		}
		for x := 0; x < g.W; x++ {
			c := g.Cells[y*g.W+x]
			if !haveStyle || c.Fg != curFg {
				fmt.Fprintf(&b, "\x1b[38;2;%d;%d;%dm", c.Fg.R, c.Fg.G, c.Fg.B)
				curFg = c.Fg
			}
			if !haveStyle || c.Bg != curBg {
				fmt.Fprintf(&b, "\x1b[48;2;%d;%d;%dm", c.Bg.R, c.Bg.G, c.Bg.B)
				curBg = c.Bg
			}
			haveStyle = true
			b.WriteRune(c.Ch)
		}
	}
	b.WriteString("\x1b[0m")
	return b.String()
}
