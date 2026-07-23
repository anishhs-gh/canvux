package render

import (
	"strings"
	"testing"

	"github.com/anishhs-gh/canvux/internal/geom"
	"github.com/anishhs-gh/canvux/internal/scene"
)

func testView(w, h int) View {
	return View{Center: geom.V(0, 0), Zoom: 2, W: w, H: h}
}

func TestViewRoundTrip(t *testing.T) {
	v := View{Center: geom.V(10, -5), Zoom: 3, W: 120, H: 80}
	p := geom.V(4.25, 7.5)
	back := v.ToWorld(v.ToPixel(p))
	if p.Dist(back) > 1e-9 {
		t.Errorf("world->pixel->world drifted: %v -> %v", p, back)
	}
}

func TestDrawLineSetsPixels(t *testing.T) {
	pb := NewPixelBuf(40, 40)
	o := &scene.Object{
		Kind: scene.KindLine, P1: geom.V(-5, -5), P2: geom.V(5, 5),
		Stroke: scene.Color{R: 255}, StrokeWidth: 1, Opacity: 1,
	}
	DrawObject(pb, testView(40, 40), o)
	c, a := pb.resolve(20, 20, scene.Color{})
	if a == 0 || c.R == 0 {
		t.Error("line should cross the center pixel")
	}
}

func TestFilledRectCoverage(t *testing.T) {
	pb := NewPixelBuf(40, 40)
	o := &scene.Object{
		Kind: scene.KindRect, P1: geom.V(-8, -8), P2: geom.V(8, 8),
		Stroke: scene.Color{R: 255}, Fill: scene.Color{G: 255}, Filled: true,
		StrokeWidth: 1, Opacity: 1,
	}
	DrawObject(pb, testView(40, 40), o)
	if _, a := pb.resolve(20, 20, scene.Color{}); a == 0 {
		t.Error("filled rect center empty")
	}
	if _, a := pb.resolve(1, 1, scene.Color{}); a != 0 {
		t.Error("outside rect should be empty")
	}
}

func TestCulling(t *testing.T) {
	pb := NewPixelBuf(10, 10)
	o := &scene.Object{
		Kind: scene.KindRect, P1: geom.V(1000, 1000), P2: geom.V(1010, 1010),
		Stroke: scene.Color{R: 255}, StrokeWidth: 1, Opacity: 1,
	}
	DrawObject(pb, testView(10, 10), o) // must not panic or set pixels
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			if _, a := pb.resolve(x, y, scene.Color{}); a != 0 {
				t.Fatal("off-screen object drew pixels")
			}
		}
	}
}

func TestCompositeModes(t *testing.T) {
	bg := scene.Color{R: 1, G: 2, B: 3}
	for _, mode := range []Mode{ModeHalfBlock, ModeBraille} {
		sx, sy := mode.PixelScale()
		pb := NewPixelBuf(4*sx, 4*sy)
		pb.Set(0, 0, scene.Color{R: 200}, 1)
		g := NewCellGrid(4, 5, bg)
		g.Composite(pb, mode, 1, bg)
		c := g.Get(0, 1)
		if c.Ch == ' ' {
			t.Errorf("%s: expected glyph in cell (0,1)", mode)
		}
		out := g.ANSI()
		if !strings.Contains(out, "\x1b[38;2;") || !strings.HasSuffix(out, "\x1b[0m") {
			t.Errorf("%s: ANSI framing broken", mode)
		}
	}
}

func TestGridStepAdaptive(t *testing.T) {
	for _, zoom := range []float64{0.05, 0.5, 2, 20, 200} {
		step := GridStep(zoom)
		px := step * zoom
		if px < 8 || px > 40 {
			t.Errorf("zoom %v: grid step %v renders at %vpx, want 8..40", zoom, step, px)
		}
	}
}

// 10k objects must rasterize a frame quickly (roadmap testing gate 9).
func BenchmarkFrame10k(b *testing.B) {
	doc := scene.NewDoc()
	for i := 0; i < 10000; i++ {
		x := float64(i%100) * 3
		y := float64(i/100) * 3
		doc.Add(&scene.Object{
			Kind: scene.KindRect, P1: geom.V(x, y), P2: geom.V(x+2, y+2),
			Stroke: scene.Color{R: uint8(i)}, StrokeWidth: 1, Opacity: 1,
		})
	}
	v := View{Center: geom.V(150, 150), Zoom: 1, W: 240, H: 120}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pb := NewPixelBuf(v.W, v.H)
		for _, o := range doc.VisibleObjects() {
			DrawObject(pb, v, o)
		}
		g := NewCellGrid(240, 62, scene.Color{})
		g.Composite(pb, ModeHalfBlock, 1, scene.Color{})
		_ = g.ANSI()
	}
}
