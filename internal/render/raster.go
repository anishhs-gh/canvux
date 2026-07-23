// Package render rasterizes the scene graph to pixel surfaces and composites
// pixels into terminal cells (half-block or braille) with ANSI output.
package render

import (
	"math"
	"sort"

	"github.com/anishhs-gh/canvux/internal/geom"
	"github.com/anishhs-gh/canvux/internal/scene"
)

// Surface is any pixel target the rasterizer can draw into.
type Surface interface {
	Size() (w, h int)
	// Set blends color c with opacity a (0..1) into pixel (x, y).
	Set(x, y int, c scene.Color, a float64)
}

// View maps world coordinates to surface pixels.
type View struct {
	Center geom.Vec // world point at the surface center
	Zoom   float64  // pixels per world unit
	W, H   int      // surface size in pixels
}

// ToPixel converts a world point to surface pixel coordinates.
func (v View) ToPixel(p geom.Vec) geom.Vec {
	return geom.V(
		(p.X-v.Center.X)*v.Zoom+float64(v.W)/2,
		(p.Y-v.Center.Y)*v.Zoom+float64(v.H)/2,
	)
}

// ToWorld converts surface pixel coordinates to a world point.
func (v View) ToWorld(p geom.Vec) geom.Vec {
	return geom.V(
		(p.X-float64(v.W)/2)/v.Zoom+v.Center.X,
		(p.Y-float64(v.H)/2)/v.Zoom+v.Center.Y,
	)
}

// WorldRect returns the visible world-space rectangle.
func (v View) WorldRect() geom.Rect {
	return geom.R(v.ToWorld(geom.V(0, 0)), v.ToWorld(geom.V(float64(v.W), float64(v.H))))
}

// DrawObject rasterizes one object into the surface.
func DrawObject(s Surface, v View, o *scene.Object) {
	if o.Kind == scene.KindText {
		return // text is composited at the cell layer, not the pixel layer
	}
	if !o.Bounds().Expand(o.StrokeWidth).Intersects(v.WorldRect()) {
		return
	}
	a := clamp01(o.Opacity)
	if fp := o.FillPolygon(); fp != nil {
		px := make([]geom.Vec, len(fp))
		for i, p := range fp {
			px[i] = v.ToPixel(p)
		}
		fillPolygon(s, px, o.Fill, a)
	}
	thick := math.Max(1, o.StrokeWidth*v.Zoom/2)
	for _, ln := range o.Outline() {
		drawPolyline(s, v, ln, o.Stroke, a, thick, o.Dashed)
	}
	if o.Kind == scene.KindArrow {
		drawArrowhead(s, v, o, a)
	}
}

func drawPolyline(s Surface, v View, pts []geom.Vec, c scene.Color, a, thick float64, dashed bool) {
	dashPhase := 0.0
	for i := 0; i+1 < len(pts); i++ {
		p1, p2 := v.ToPixel(pts[i]), v.ToPixel(pts[i+1])
		dashPhase = drawLine(s, p1, p2, c, a, thick, dashed, dashPhase)
	}
}

// drawLine draws a thick, optionally dashed line in pixel space and returns
// the updated dash phase so dashes flow continuously across joints.
func drawLine(s Surface, p1, p2 geom.Vec, c scene.Color, a, thick float64, dashed bool, phase float64) float64 {
	const dashOn, dashPeriod = 4.0, 8.0
	length := p1.Dist(p2)
	steps := int(length) + 1
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		if dashed {
			d := math.Mod(phase+length*t, dashPeriod)
			if d > dashOn {
				continue
			}
		}
		p := p1.Lerp(p2, t)
		stamp(s, p, c, a, thick)
	}
	return math.Mod(phase+length, dashPeriod)
}

// stamp draws a filled disc of radius r at pixel p.
func stamp(s Surface, p geom.Vec, c scene.Color, a, r float64) {
	if r <= 0.75 {
		s.Set(int(math.Round(p.X)), int(math.Round(p.Y)), c, a)
		return
	}
	ir := int(math.Ceil(r))
	for dy := -ir; dy <= ir; dy++ {
		for dx := -ir; dx <= ir; dx++ {
			if float64(dx*dx+dy*dy) <= r*r {
				s.Set(int(math.Round(p.X))+dx, int(math.Round(p.Y))+dy, c, a)
			}
		}
	}
}

// fillPolygon scanline-fills a polygon (even-odd rule) in pixel space.
func fillPolygon(s Surface, pts []geom.Vec, c scene.Color, a float64) {
	if len(pts) < 3 {
		return
	}
	_, h := s.Size()
	minY, maxY := math.Inf(1), math.Inf(-1)
	for _, p := range pts {
		minY, maxY = math.Min(minY, p.Y), math.Max(maxY, p.Y)
	}
	y0 := int(math.Max(0, math.Floor(minY)))
	y1 := int(math.Min(float64(h-1), math.Ceil(maxY)))
	var xs []float64
	for y := y0; y <= y1; y++ {
		fy := float64(y) + 0.5
		xs = xs[:0]
		for i := range pts {
			p1, p2 := pts[i], pts[(i+1)%len(pts)]
			if (p1.Y <= fy) != (p2.Y <= fy) {
				t := (fy - p1.Y) / (p2.Y - p1.Y)
				xs = append(xs, p1.X+t*(p2.X-p1.X))
			}
		}
		sort.Float64s(xs)
		for i := 0; i+1 < len(xs); i += 2 {
			for x := int(math.Round(xs[i])); x <= int(math.Round(xs[i+1])); x++ {
				s.Set(x, y, c, a)
			}
		}
	}
}

func drawArrowhead(s Surface, v View, o *scene.Object, a float64) {
	p1, p2 := o.P1, o.P2
	if o.Rotation != 0 {
		c := o.Center()
		p1, p2 = p1.RotateAround(c, o.Rotation), p2.RotateAround(c, o.Rotation)
	}
	dir := p2.Sub(p1)
	if dir.Len() == 0 {
		return
	}
	n := dir.Mul(1 / dir.Len())
	// World-unit head size, matching the SVG exporter.
	size := math.Max(1.5, math.Min(3, dir.Len()*0.3))
	base := p2.Sub(n.Mul(size))
	perp := geom.V(-n.Y, n.X).Mul(size * 0.5)
	tip := []geom.Vec{v.ToPixel(p2), v.ToPixel(base.Add(perp)), v.ToPixel(base.Sub(perp))}
	fillPolygon(s, tip, o.Stroke, a)
}

// DrawGrid renders adaptive grid dots and the world origin axes.
func DrawGrid(s Surface, v View, dot, axis scene.Color) {
	// Pick a grid step that lands between 8 and 40 px on screen: 1,2,5 x 10^n.
	step := 1.0
	for step*v.Zoom < 8 {
		step *= 2
		if step*v.Zoom < 8 {
			step *= 2.5
		}
	}
	for step*v.Zoom > 40 {
		step /= 2
		if step*v.Zoom > 40 {
			step /= 2.5
		}
	}
	wr := v.WorldRect()
	x0 := math.Floor(wr.Min.X/step) * step
	y0 := math.Floor(wr.Min.Y/step) * step
	for y := y0; y <= wr.Max.Y; y += step {
		for x := x0; x <= wr.Max.X; x += step {
			p := v.ToPixel(geom.V(x, y))
			s.Set(int(p.X), int(p.Y), dot, 1)
		}
	}
	// Origin axes as faint lines.
	origin := v.ToPixel(geom.V(0, 0))
	w, h := s.Size()
	if origin.X >= 0 && origin.X < float64(w) {
		for y := 0; y < h; y += 2 {
			s.Set(int(origin.X), y, axis, 1)
		}
	}
	if origin.Y >= 0 && origin.Y < float64(h) {
		for x := 0; x < w; x += 2 {
			s.Set(x, int(origin.Y), axis, 1)
		}
	}
}

// GridStep exposes the adaptive step used by DrawGrid for snapping.
func GridStep(zoom float64) float64 {
	step := 1.0
	for step*zoom < 8 {
		step *= 2
		if step*zoom < 8 {
			step *= 2.5
		}
	}
	for step*zoom > 40 {
		step /= 2
		if step*zoom > 40 {
			step /= 2.5
		}
	}
	return step
}

func clamp01(f float64) float64 { return math.Max(0, math.Min(1, f)) }
