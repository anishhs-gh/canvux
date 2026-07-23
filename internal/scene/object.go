// Package scene implements the Canvux scene graph: objects, layers,
// the document, hit testing and (de)serialization.
package scene

import (
	"fmt"
	"math"

	"github.com/anishhs-gh/canvux/internal/geom"
)

// Kind enumerates the vector object types Canvux understands.
type Kind string

const (
	KindLine    Kind = "line"
	KindRect    Kind = "rect"
	KindEllipse Kind = "ellipse"
	KindPolygon Kind = "polygon"
	KindPath    Kind = "path" // freehand brush stroke (open polyline)
	KindArrow   Kind = "arrow"
	KindText    Kind = "text"
)

// Color is an RGB color serialized as "#rrggbb".
type Color struct{ R, G, B uint8 }

func (c Color) Hex() string { return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B) }

func (c Color) MarshalText() ([]byte, error) { return []byte(c.Hex()), nil }

func (c *Color) UnmarshalText(b []byte) error {
	s := string(b)
	if len(s) == 7 && s[0] == '#' {
		var r, g, bl uint8
		if _, err := fmt.Sscanf(s, "#%02x%02x%02x", &r, &g, &bl); err == nil {
			*c = Color{r, g, bl}
			return nil
		}
	}
	return fmt.Errorf("invalid color %q", s)
}

// Object is a single vector element on the canvas.
//
// Geometry storage by kind:
//   - line/arrow:   P1 -> P2
//   - rect/ellipse: bounding box R(P1, P2)
//   - polygon/path: Points
//   - text:         anchored at P1
type Object struct {
	ID     uint64     `json:"id"`
	Kind   Kind       `json:"kind"`
	P1     geom.Vec   `json:"p1,omitempty"`
	P2     geom.Vec   `json:"p2,omitempty"`
	Points []geom.Vec `json:"points,omitempty"`
	Text   string     `json:"text,omitempty"`

	Stroke      Color   `json:"stroke"`
	Fill        Color   `json:"fill"`
	Filled      bool    `json:"filled,omitempty"`
	StrokeWidth float64 `json:"strokeWidth"`
	Opacity     float64 `json:"opacity"`
	Dashed      bool    `json:"dashed,omitempty"`
	Rotation    float64 `json:"rotation,omitempty"` // radians about the bounds center
	Layer       int     `json:"layer"`
}

// Clone returns a deep copy of the object.
func (o *Object) Clone() *Object {
	c := *o
	if o.Points != nil {
		c.Points = append([]geom.Vec(nil), o.Points...)
	}
	return &c
}

// baseBounds is the unrotated bounding box.
func (o *Object) baseBounds() geom.Rect {
	switch o.Kind {
	case KindPolygon, KindPath:
		return geom.BoundsOf(o.Points)
	case KindText:
		// One cell per rune; text renders at fixed terminal size but we give it
		// a nominal world footprint so it can be selected and framed.
		w := float64(len([]rune(o.Text)))
		if w < 1 {
			w = 1
		}
		return geom.Rect{Min: o.P1, Max: o.P1.Add(geom.V(w, 2))}
	default:
		return geom.R(o.P1, o.P2)
	}
}

// Bounds returns the world-space axis-aligned bounding box, including rotation.
func (o *Object) Bounds() geom.Rect {
	b := o.baseBounds()
	if o.Rotation == 0 {
		return b
	}
	c := b.Center()
	corners := b.Corners()
	pts := make([]geom.Vec, 4)
	for i, p := range corners {
		pts[i] = p.RotateAround(c, o.Rotation)
	}
	return geom.BoundsOf(pts)
}

// Center returns the rotation pivot (center of the unrotated bounds).
func (o *Object) Center() geom.Vec { return o.baseBounds().Center() }

// Outline returns the object's geometry as one or more polylines in world
// space with rotation applied. Closed shapes repeat their first point.
func (o *Object) Outline() [][]geom.Vec {
	var lines [][]geom.Vec
	switch o.Kind {
	case KindLine, KindArrow:
		lines = [][]geom.Vec{{o.P1, o.P2}}
	case KindRect:
		c := geom.R(o.P1, o.P2).Corners()
		lines = [][]geom.Vec{{c[0], c[1], c[2], c[3], c[0]}}
	case KindEllipse:
		lines = [][]geom.Vec{ellipsePoints(geom.R(o.P1, o.P2), 48)}
	case KindPolygon:
		if len(o.Points) > 0 {
			ring := append(append([]geom.Vec(nil), o.Points...), o.Points[0])
			lines = [][]geom.Vec{ring}
		}
	case KindPath:
		lines = [][]geom.Vec{append([]geom.Vec(nil), o.Points...)}
	case KindText:
		c := o.baseBounds().Corners()
		lines = [][]geom.Vec{{c[0], c[1], c[2], c[3], c[0]}}
	}
	if o.Rotation != 0 {
		p := o.Center()
		for _, ln := range lines {
			for i := range ln {
				ln[i] = ln[i].RotateAround(p, o.Rotation)
			}
		}
	}
	return lines
}

// FillPolygon returns the closed polygon to fill, or nil if the object is not
// filled. Rotation is applied.
func (o *Object) FillPolygon() []geom.Vec {
	if !o.Filled {
		return nil
	}
	var poly []geom.Vec
	switch o.Kind {
	case KindRect:
		c := geom.R(o.P1, o.P2).Corners()
		poly = c[:]
	case KindEllipse:
		poly = ellipsePoints(geom.R(o.P1, o.P2), 48)
	case KindPolygon:
		poly = append([]geom.Vec(nil), o.Points...)
	default:
		return nil
	}
	if o.Rotation != 0 {
		p := o.Center()
		for i := range poly {
			poly[i] = poly[i].RotateAround(p, o.Rotation)
		}
	}
	return poly
}

// Hit reports whether world point p touches the object within tolerance tol.
func (o *Object) Hit(p geom.Vec, tol float64) bool {
	if !o.Bounds().Expand(tol).Contains(p) {
		return false
	}
	if o.Kind == KindText {
		return true // bounds hit is enough for text
	}
	if fp := o.FillPolygon(); fp != nil && geom.PointInPolygon(p, fp) {
		return true
	}
	tol = math.Max(tol, o.StrokeWidth/2)
	for _, ln := range o.Outline() {
		for i := 0; i+1 < len(ln); i++ {
			if geom.DistToSegment(p, ln[i], ln[i+1]) <= tol {
				return true
			}
		}
	}
	return false
}

// Translate moves the object by d.
func (o *Object) Translate(d geom.Vec) {
	o.P1 = o.P1.Add(d)
	o.P2 = o.P2.Add(d)
	for i := range o.Points {
		o.Points[i] = o.Points[i].Add(d)
	}
}

// ScaleAround scales the object's geometry about pivot by (sx, sy).
func (o *Object) ScaleAround(pivot geom.Vec, sx, sy float64) {
	scale := func(p geom.Vec) geom.Vec {
		return geom.V(pivot.X+(p.X-pivot.X)*sx, pivot.Y+(p.Y-pivot.Y)*sy)
	}
	o.P1 = scale(o.P1)
	o.P2 = scale(o.P2)
	for i := range o.Points {
		o.Points[i] = scale(o.Points[i])
	}
}

func ellipsePoints(b geom.Rect, n int) []geom.Vec {
	c := b.Center()
	rx, ry := b.W()/2, b.H()/2
	pts := make([]geom.Vec, n)
	for i := 0; i < n; i++ {
		a := 2 * math.Pi * float64(i) / float64(n)
		pts[i] = geom.V(c.X+rx*math.Cos(a), c.Y+ry*math.Sin(a))
	}
	return pts
}
