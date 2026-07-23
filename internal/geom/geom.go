// Package geom provides 2D vector math primitives for the Canvux scene graph.
package geom

import "math"

// Vec is a 2D point or vector in world coordinates.
type Vec struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

func V(x, y float64) Vec { return Vec{x, y} }

func (a Vec) Add(b Vec) Vec      { return Vec{a.X + b.X, a.Y + b.Y} }
func (a Vec) Sub(b Vec) Vec      { return Vec{a.X - b.X, a.Y - b.Y} }
func (a Vec) Mul(s float64) Vec  { return Vec{a.X * s, a.Y * s} }
func (a Vec) Dot(b Vec) float64  { return a.X*b.X + a.Y*b.Y }
func (a Vec) Len() float64       { return math.Hypot(a.X, a.Y) }
func (a Vec) Dist(b Vec) float64 { return a.Sub(b).Len() }

// Rotate rotates the vector by angle radians around the origin.
func (a Vec) Rotate(angle float64) Vec {
	s, c := math.Sincos(angle)
	return Vec{a.X*c - a.Y*s, a.X*s + a.Y*c}
}

// RotateAround rotates the point by angle radians around pivot p.
func (a Vec) RotateAround(p Vec, angle float64) Vec {
	return a.Sub(p).Rotate(angle).Add(p)
}

// Lerp linearly interpolates between a and b by t in [0,1].
func (a Vec) Lerp(b Vec, t float64) Vec {
	return Vec{a.X + (b.X-a.X)*t, a.Y + (b.Y-a.Y)*t}
}

// Rect is an axis-aligned rectangle. Min is the top-left, Max the bottom-right.
type Rect struct {
	Min Vec `json:"min"`
	Max Vec `json:"max"`
}

// R returns a normalized rectangle from any two corner points.
func R(a, b Vec) Rect {
	return Rect{
		Min: Vec{math.Min(a.X, b.X), math.Min(a.Y, b.Y)},
		Max: Vec{math.Max(a.X, b.X), math.Max(a.Y, b.Y)},
	}
}

func (r Rect) W() float64    { return r.Max.X - r.Min.X }
func (r Rect) H() float64    { return r.Max.Y - r.Min.Y }
func (r Rect) Center() Vec   { return Vec{(r.Min.X + r.Max.X) / 2, (r.Min.Y + r.Max.Y) / 2} }
func (r Rect) IsEmpty() bool { return r.Max.X <= r.Min.X && r.Max.Y <= r.Min.Y }
func (r Rect) Contains(p Vec) bool {
	return p.X >= r.Min.X && p.X <= r.Max.X && p.Y >= r.Min.Y && p.Y <= r.Max.Y
}

// Expand grows the rectangle by d on every side.
func (r Rect) Expand(d float64) Rect {
	return Rect{Vec{r.Min.X - d, r.Min.Y - d}, Vec{r.Max.X + d, r.Max.Y + d}}
}

// Union returns the smallest rectangle covering both r and o.
func (r Rect) Union(o Rect) Rect {
	return Rect{
		Min: Vec{math.Min(r.Min.X, o.Min.X), math.Min(r.Min.Y, o.Min.Y)},
		Max: Vec{math.Max(r.Max.X, o.Max.X), math.Max(r.Max.Y, o.Max.Y)},
	}
}

// Intersects reports whether r and o overlap.
func (r Rect) Intersects(o Rect) bool {
	return r.Min.X <= o.Max.X && o.Min.X <= r.Max.X && r.Min.Y <= o.Max.Y && o.Min.Y <= r.Max.Y
}

// Corners returns the four corners in clockwise order from top-left.
func (r Rect) Corners() [4]Vec {
	return [4]Vec{r.Min, {r.Max.X, r.Min.Y}, r.Max, {r.Min.X, r.Max.Y}}
}

// BoundsOf returns the bounding rectangle of a set of points.
func BoundsOf(pts []Vec) Rect {
	if len(pts) == 0 {
		return Rect{}
	}
	b := Rect{pts[0], pts[0]}
	for _, p := range pts[1:] {
		b.Min.X = math.Min(b.Min.X, p.X)
		b.Min.Y = math.Min(b.Min.Y, p.Y)
		b.Max.X = math.Max(b.Max.X, p.X)
		b.Max.Y = math.Max(b.Max.Y, p.Y)
	}
	return b
}

// DistToSegment returns the distance from point p to segment a-b.
func DistToSegment(p, a, b Vec) float64 {
	ab := b.Sub(a)
	l2 := ab.Dot(ab)
	if l2 == 0 {
		return p.Dist(a)
	}
	t := math.Max(0, math.Min(1, p.Sub(a).Dot(ab)/l2))
	return p.Dist(a.Add(ab.Mul(t)))
}

// PointInPolygon reports whether p is inside the polygon (even-odd rule).
func PointInPolygon(p Vec, poly []Vec) bool {
	inside := false
	n := len(poly)
	for i, j := 0, n-1; i < n; j, i = i, i+1 {
		a, b := poly[i], poly[j]
		if (a.Y > p.Y) != (b.Y > p.Y) &&
			p.X < (b.X-a.X)*(p.Y-a.Y)/(b.Y-a.Y)+a.X {
			inside = !inside
		}
	}
	return inside
}
