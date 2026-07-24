package app

import (
	"math"

	"github.com/anishhs-gh/canvux/internal/geom"
)

// guideLine is a world-space alignment guide drawn during a drag.
type guideLine struct {
	a, b geom.Vec
}

// alignResult carries the snap adjustment and the guides to draw for it.
type alignResult struct {
	dx, dy float64
	guides []guideLine
}

// alignSnap computes an alignment adjustment for a moving rectangle against the
// edges and centers of other (non-excluded) objects. When a candidate's
// left/center/right x (or top/center/bottom y) falls within tol world units of
// the moving box's corresponding line, it snaps and emits a guide. tol is in
// world units (derive from a few screen pixels).
func (m *Model) alignSnap(moving geom.Rect, exclude map[uint64]bool, tol float64) alignResult {
	var res alignResult
	bestX, bestY := tol, tol // only snap within tolerance; track the closest
	haveX, haveY := false, false
	var gx, gy guideLine

	mxs := [3]float64{moving.Min.X, moving.Center().X, moving.Max.X}
	mys := [3]float64{moving.Min.Y, moving.Center().Y, moving.Max.Y}

	for _, o := range m.doc.VisibleObjects() {
		if exclude[o.ID] {
			continue
		}
		b := o.Bounds()
		oxs := [3]float64{b.Min.X, b.Center().X, b.Max.X}
		oys := [3]float64{b.Min.Y, b.Center().Y, b.Max.Y}

		for _, mx := range mxs {
			for _, ox := range oxs {
				if d := math.Abs(mx - ox); d < bestX {
					bestX, haveX = d, true
					res.dx = ox - mx
					// Vertical guide spanning both boxes.
					y0 := math.Min(moving.Min.Y, b.Min.Y)
					y1 := math.Max(moving.Max.Y, b.Max.Y)
					gx = guideLine{geom.V(ox, y0), geom.V(ox, y1)}
				}
			}
		}
		for _, my := range mys {
			for _, oy := range oys {
				if d := math.Abs(my - oy); d < bestY {
					bestY, haveY = d, true
					res.dy = oy - my
					x0 := math.Min(moving.Min.X, b.Min.X)
					x1 := math.Max(moving.Max.X, b.Max.X)
					gy = guideLine{geom.V(x0, oy), geom.V(x1, oy)}
				}
			}
		}
	}
	if haveX {
		res.guides = append(res.guides, gx)
	} else {
		res.dx = 0
	}
	if haveY {
		res.guides = append(res.guides, gy)
	} else {
		res.dy = 0
	}
	return res
}

// snapPointToObjects nudges a single point onto the nearest object edge/center
// x and y within tol, returning the adjusted point and the guides to draw.
// Used while drawing so a shape's active corner aligns to existing objects.
func (m *Model) snapPointToObjects(p geom.Vec, exclude map[uint64]bool, tol float64) (geom.Vec, []guideLine) {
	bestX, bestY := tol, tol
	haveX, haveY := false, false
	var gx, gy guideLine
	out := p
	for _, o := range m.doc.VisibleObjects() {
		if exclude[o.ID] {
			continue
		}
		b := o.Bounds()
		for _, ox := range [3]float64{b.Min.X, b.Center().X, b.Max.X} {
			if d := math.Abs(p.X - ox); d < bestX {
				bestX, haveX, out.X = d, true, ox
				gx = guideLine{geom.V(ox, math.Min(p.Y, b.Min.Y)), geom.V(ox, math.Max(p.Y, b.Max.Y))}
			}
		}
		for _, oy := range [3]float64{b.Min.Y, b.Center().Y, b.Max.Y} {
			if d := math.Abs(p.Y - oy); d < bestY {
				bestY, haveY, out.Y = d, true, oy
				gy = guideLine{geom.V(math.Min(p.X, b.Min.X), oy), geom.V(math.Max(p.X, b.Max.X), oy)}
			}
		}
	}
	var guides []guideLine
	if haveX {
		guides = append(guides, gx)
	}
	if haveY {
		guides = append(guides, gy)
	}
	return out, guides
}

// alignTolerance is the alignment snap distance in world units (~5 px).
func (m *Model) alignTolerance() float64 { return 5 / m.view().Zoom }
