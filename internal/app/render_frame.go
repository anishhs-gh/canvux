package app

import (
	"math"

	"github.com/anishhs-gh/canvux/internal/render"
	"github.com/anishhs-gh/canvux/internal/scene"
)

// RenderFrame rasterizes one non-interactive frame of the document, fitted to
// its content, for `canvux render` (printing a drawing straight to stdout).
// Output color is degraded to the given profile (use render.DetectProfile()).
func RenderFrame(doc *scene.Doc, cols, rows int, mode render.Mode, profile render.Profile) string {
	t := DefaultTheme
	sx, sy := mode.PixelScale()
	v := render.View{W: cols * sx, H: rows * sy, Zoom: 2}

	if b := doc.ContentBounds(); b.W() > 0 || b.H() > 0 {
		b = b.Expand(2)
		zx := float64(v.W) / math.Max(b.W(), 0.001)
		zy := float64(v.H) / math.Max(b.H(), 0.001)
		v.Zoom = math.Min(zx, zy)
		v.Center = b.Center()
	}

	pb := render.NewPixelBuf(v.W, v.H)
	for _, o := range doc.VisibleObjects() {
		render.DrawObject(pb, v, o)
	}
	g := render.NewCellGrid(cols, rows, t.CanvasBG)
	g.Profile = profile
	g.Composite(pb, mode, 0, t.CanvasBG)

	// Text objects at the cell layer.
	for _, o := range doc.VisibleObjects() {
		if o.Kind != scene.KindText {
			continue
		}
		p := v.ToPixel(o.P1)
		cx, cy := int(p.X)/sx, int(p.Y)/sy
		for i, r := range o.Text {
			under := g.Get(cx+i, cy)
			g.Set(cx+i, cy, render.Cell{Ch: r, Fg: o.Stroke, Bg: under.Bg})
		}
	}
	return g.ANSI()
}
