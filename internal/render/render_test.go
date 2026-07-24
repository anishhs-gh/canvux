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

func TestGradientFillVaries(t *testing.T) {
	pb := NewPixelBuf(40, 40)
	f2 := scene.Color{R: 255}
	o := &scene.Object{
		Kind: scene.KindRect, P1: geom.V(-9, -9), P2: geom.V(9, 9),
		Stroke: scene.Color{}, Fill: scene.Color{B: 255}, Fill2: &f2,
		Filled: true, StrokeWidth: 1, Opacity: 1, GradAngle: 90,
	}
	DrawObject(pb, testView(40, 40), o)
	top, _ := pb.resolve(20, 5, scene.Color{})
	bot, _ := pb.resolve(20, 35, scene.Color{})
	if top.B < bot.B || bot.R < top.R {
		t.Errorf("vertical gradient not blue->red: top=%+v bot=%+v", top, bot)
	}
	if top == bot {
		t.Error("gradient fill produced a flat color")
	}
}

func TestShadowDrawsOffsetPixels(t *testing.T) {
	pb := NewPixelBuf(60, 60)
	o := &scene.Object{
		Kind: scene.KindRect, P1: geom.V(-5, -5), P2: geom.V(5, 5),
		Stroke: scene.Color{R: 255}, Fill: scene.Color{R: 255}, Filled: true,
		StrokeWidth: 1, Opacity: 1, Shadow: true,
	}
	DrawObject(pb, testView(60, 60), o)
	// Beyond the bottom-right corner (shape edge at px 40): shadow territory.
	if _, a := pb.resolve(42, 42, scene.Color{}); a == 0 {
		t.Error("no shadow pixels below-right of the shape")
	}
	if _, a := pb.resolve(14, 14, scene.Color{}); a != 0 {
		t.Error("shadow must not appear above-left of the shape")
	}
}

func TestBlurSpreadsPixels(t *testing.T) {
	sharp := NewPixelBuf(60, 60)
	blurred := NewPixelBuf(60, 60)
	base := &scene.Object{
		Kind: scene.KindRect, P1: geom.V(-5, -5), P2: geom.V(5, 5),
		Stroke: scene.Color{G: 255}, StrokeWidth: 1, Opacity: 1,
	}
	DrawObject(sharp, testView(60, 60), base)
	b := base.Clone()
	b.Blur = 2
	DrawObject(blurred, testView(60, 60), b)
	count := func(p *PixelBuf) int {
		n := 0
		for y := 0; y < 60; y++ {
			for x := 0; x < 60; x++ {
				if _, a := p.resolve(x, y, scene.Color{}); a > 0.05 {
					n++
				}
			}
		}
		return n
	}
	if count(blurred) <= count(sharp) {
		t.Error("blur should cover more pixels than the sharp original")
	}
}

func TestVariableWidthStroke(t *testing.T) {
	pb := NewPixelBuf(80, 40)
	o := &scene.Object{
		Kind:   scene.KindPath,
		Points: []geom.Vec{{X: -15, Y: 0}, {X: 0, Y: 0}, {X: 15, Y: 0}},
		Widths: []float64{0.5, 0.5, 6},
		Stroke: scene.Color{R: 255}, StrokeWidth: 1, Opacity: 1,
	}
	DrawObject(pb, testView(80, 40), o)
	// Thick end should paint far more vertical extent than the thin end.
	colCount := func(x int) int {
		n := 0
		for y := 0; y < 40; y++ {
			if _, a := pb.resolve(x, y, scene.Color{}); a > 0 {
				n++
			}
		}
		return n
	}
	if thin, thick := colCount(14), colCount(66); thick <= thin+2 {
		t.Errorf("variable width flat: thin col=%d thick col=%d", thin, thick)
	}
}

func TestPixelBufResizeReuses(t *testing.T) {
	pb := NewPixelBuf(10, 10)
	pb.Set(5, 5, scene.Color{R: 200}, 1)
	before := &pb.pix[0]
	// Resize to a smaller/equal size should reuse the backing array and clear.
	pb.Resize(8, 8)
	if pb.W != 8 || pb.H != 8 {
		t.Fatalf("resize dims = %dx%d", pb.W, pb.H)
	}
	if &pb.pix[0] != before {
		t.Error("Resize reallocated when it could reuse")
	}
	if _, a := pb.resolve(5, 5, scene.Color{}); a != 0 {
		t.Error("Resize did not clear existing pixels")
	}
	// Growing beyond capacity must allocate.
	pb.Resize(100, 100)
	if pb.W != 100 || len(pb.pix) != 10000 {
		t.Errorf("grow failed: %dx%d len=%d", pb.W, pb.H, len(pb.pix))
	}
}

func TestCellGridResetReuses(t *testing.T) {
	bg := scene.Color{R: 1, G: 2, B: 3}
	g := NewCellGrid(10, 4, bg)
	g.Set(0, 0, Cell{Ch: 'X', Fg: scene.Color{R: 9}, Bg: bg})
	before := &g.Cells[0]
	g.Reset(8, 4, bg)
	if g.W != 8 || g.H != 4 {
		t.Fatalf("reset dims = %dx%d", g.W, g.H)
	}
	if &g.Cells[0] != before {
		t.Error("Reset reallocated when it could reuse")
	}
	if g.Get(0, 0).Ch != ' ' {
		t.Error("Reset did not clear cells")
	}
	// A reset grid renders identically to a fresh one.
	fresh := NewCellGrid(8, 4, bg)
	fresh.Profile, g.Profile = TrueColor, TrueColor
	if g.ANSI() != fresh.ANSI() {
		t.Error("reset grid differs from fresh grid")
	}
}

// Demonstrates the per-frame allocation win from reusing buffers.
func BenchmarkFrameFreshAlloc(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		pb := NewPixelBuf(240, 120)
		g := NewCellGrid(240, 62, scene.Color{})
		pb.Set(1, 1, scene.Color{R: 200}, 1)
		g.Composite(pb, ModeHalfBlock, 1, scene.Color{})
	}
}

func BenchmarkFrameReuseBuffers(b *testing.B) {
	b.ReportAllocs()
	pb := NewPixelBuf(240, 120)
	g := NewCellGrid(240, 62, scene.Color{})
	for i := 0; i < b.N; i++ {
		pb.Resize(240, 120)
		g.Reset(240, 62, scene.Color{})
		pb.Set(1, 1, scene.Color{R: 200}, 1)
		g.Composite(pb, ModeHalfBlock, 1, scene.Color{})
	}
}
