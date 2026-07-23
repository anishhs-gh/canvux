package scene

import (
	"testing"

	"github.com/anishhs-gh/canvux/internal/geom"
)

func newTestObject(kind Kind) *Object {
	return &Object{
		Kind: kind, Stroke: Color{R: 255}, StrokeWidth: 1, Opacity: 1,
		P1: geom.V(0, 0), P2: geom.V(10, 10),
	}
}

func TestJSONRoundTrip(t *testing.T) {
	d := NewDoc()
	r := newTestObject(KindRect)
	r.Filled = true
	r.Fill = Color{G: 128}
	r.Rotation = 0.5
	d.Add(r)
	p := newTestObject(KindPath)
	p.Points = []geom.Vec{{X: 1, Y: 2}, {X: 3, Y: 4}, {X: 5, Y: 0}}
	d.Add(p)
	txt := newTestObject(KindText)
	txt.Text = "hello <world> & more"
	d.Add(txt)

	data, err := d.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	got, err := Unmarshal(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Objects) != 3 {
		t.Fatalf("objects = %d, want 3", len(got.Objects))
	}
	if got.Objects[0].Fill != (Color{G: 128}) || !got.Objects[0].Filled {
		t.Errorf("fill lost: %+v", got.Objects[0])
	}
	if got.Objects[2].Text != "hello <world> & more" {
		t.Errorf("text lost: %q", got.Objects[2].Text)
	}
	// IDs must keep incrementing after load (no collisions).
	next := newTestObject(KindLine)
	got.Add(next)
	for _, o := range got.Objects[:3] {
		if o.ID == next.ID {
			t.Errorf("duplicate ID %d after reload", next.ID)
		}
	}
}

func TestHitTest(t *testing.T) {
	d := NewDoc()
	r := newTestObject(KindRect)
	d.Add(r)
	if d.HitTest(geom.V(5, 0), 0.5) == nil {
		t.Error("expected hit on rect edge")
	}
	if d.HitTest(geom.V(5, 5), 0.5) != nil {
		t.Error("unfilled rect center should not hit")
	}
	r.Filled = true
	if d.HitTest(geom.V(5, 5), 0.5) == nil {
		t.Error("filled rect center should hit")
	}
	if d.HitTest(geom.V(50, 50), 0.5) != nil {
		t.Error("far point should miss")
	}
}

func TestHitTestRespectsLayers(t *testing.T) {
	d := NewDoc()
	d.Layers = append(d.Layers, Layer{Name: "top", Visible: true})
	r := newTestObject(KindRect)
	r.Filled = true
	d.Add(r)
	d.Layers[0].Visible = false
	if d.HitTest(geom.V(5, 5), 0.5) != nil {
		t.Error("hidden layer object should not hit")
	}
	d.Layers[0].Visible = true
	d.Layers[0].Locked = true
	if d.HitTest(geom.V(5, 5), 0.5) != nil {
		t.Error("locked layer object should not hit")
	}
}

func TestZOrder(t *testing.T) {
	d := NewDoc()
	a, b := newTestObject(KindRect), newTestObject(KindRect)
	d.Add(a)
	d.Add(b)
	d.Raise(a.ID)
	if d.Objects[1].ID != a.ID {
		t.Error("raise failed")
	}
	d.Lower(a.ID)
	if d.Objects[0].ID != a.ID {
		t.Error("lower failed")
	}
}

func TestRotatedBounds(t *testing.T) {
	o := newTestObject(KindRect)
	o.Rotation = 0.7854 // ~45°
	b := o.Bounds()
	if b.W() < 13 || b.W() > 15 {
		t.Errorf("rotated 10x10 rect width = %f, want ~14.14", b.W())
	}
}

func TestCloneIsDeep(t *testing.T) {
	o := newTestObject(KindPath)
	o.Points = []geom.Vec{{X: 1, Y: 1}}
	c := o.Clone()
	c.Points[0].X = 99
	if o.Points[0].X == 99 {
		t.Error("Clone shares Points slice")
	}
}

func TestVersionGuard(t *testing.T) {
	if _, err := Unmarshal([]byte(`{"version": 999}`)); err == nil {
		t.Error("expected error for future format version")
	}
}

func TestBezier(t *testing.T) {
	o := &Object{
		Kind: KindBezier, Stroke: Color{R: 255}, StrokeWidth: 1, Opacity: 1,
		P1: geom.V(0, 0), C1: geom.V(0, 10), C2: geom.V(10, 10), P2: geom.V(10, 0),
	}
	pts := o.BezierPoints(24)
	if pts[0] != o.P1 || pts[24] != o.P2 {
		t.Error("bezier endpoints wrong")
	}
	b := o.Bounds()
	if b.Max.Y < 5 || b.Max.Y > 8 {
		t.Errorf("bezier bulge bounds = %v, want max.Y ~7.5", b)
	}
	// The curve passes near (5, 7.5) at t=0.5.
	if !o.Hit(geom.V(5, 7.5), 0.6) {
		t.Error("expected hit on curve midpoint")
	}
	if o.Hit(geom.V(5, 0), 0.4) {
		t.Error("chord midpoint should not hit the bowed curve")
	}
	// Translate must carry control points.
	o.Translate(geom.V(100, 0))
	if o.C1.X != 100 {
		t.Errorf("Translate ignored C1: %v", o.C1)
	}
}

func TestCloneDeepCopiesPhase9Fields(t *testing.T) {
	f2 := Color{G: 9}
	o := &Object{Kind: KindPath, Points: []geom.Vec{{X: 1}}, Widths: []float64{2}, Fill2: &f2}
	c := o.Clone()
	c.Widths[0] = 99
	c.Fill2.G = 99
	if o.Widths[0] == 99 || o.Fill2.G == 99 {
		t.Error("Clone shares Widths or Fill2")
	}
}

func TestUnmarshalDropsMismatchedWidths(t *testing.T) {
	d := NewDoc()
	o := &Object{Kind: KindPath, Points: []geom.Vec{{X: 1}, {X: 2}}, Widths: []float64{1},
		Stroke: Color{}, StrokeWidth: 1, Opacity: 1}
	d.Add(o)
	data, _ := d.Marshal()
	got, err := Unmarshal(data)
	if err != nil {
		t.Fatal(err)
	}
	if got.Objects[0].Widths != nil {
		t.Error("mismatched widths should be dropped on load")
	}
}
