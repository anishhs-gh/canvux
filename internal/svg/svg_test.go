package svg

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/anishhs-gh/canvux/internal/geom"
	"github.com/anishhs-gh/canvux/internal/scene"
)

func TestExportIsValidXML(t *testing.T) {
	d := scene.NewDoc()
	for _, k := range []scene.Kind{scene.KindLine, scene.KindRect, scene.KindEllipse, scene.KindArrow} {
		o := &scene.Object{
			Kind: k, P1: geom.V(0, 0), P2: geom.V(10, 8),
			Stroke: scene.Color{R: 255}, StrokeWidth: 1, Opacity: 1,
		}
		d.Add(o)
	}
	poly := &scene.Object{
		Kind:   scene.KindPolygon,
		Points: []geom.Vec{{X: 0, Y: 0}, {X: 5, Y: 0}, {X: 3, Y: 4}},
		Stroke: scene.Color{G: 255}, Fill: scene.Color{B: 255}, Filled: true,
		StrokeWidth: 1, Opacity: 0.5, Dashed: true, Rotation: 0.3,
	}
	d.Add(poly)
	txt := &scene.Object{
		Kind: scene.KindText, P1: geom.V(1, 1), Text: "a<b>&c",
		Stroke: scene.Color{R: 200}, StrokeWidth: 1, Opacity: 1,
	}
	d.Add(txt)
	path := &scene.Object{
		Kind:   scene.KindPath,
		Points: []geom.Vec{{X: 0, Y: 0}, {X: 2, Y: 1}, {X: 4, Y: 0}},
		Stroke: scene.Color{R: 128}, StrokeWidth: 1, Opacity: 1,
	}
	d.Add(path)

	out := Export(d)
	dec := xml.NewDecoder(strings.NewReader(string(out)))
	for {
		if _, err := dec.Token(); err != nil {
			if err.Error() == "EOF" {
				break
			}
			t.Fatalf("export produced invalid XML: %v\n%s", err, out)
		}
	}
	for _, want := range []string{"<line", "<rect", "<ellipse", "<polygon", "<polyline", "<text", "stroke-dasharray", "rotate("} {
		if !strings.Contains(string(out), want) {
			t.Errorf("export missing %q", want)
		}
	}
}

func TestRoundTrip(t *testing.T) {
	d := scene.NewDoc()
	r := &scene.Object{
		Kind: scene.KindRect, P1: geom.V(2, 3), P2: geom.V(12, 9),
		Stroke: scene.Color{R: 0xaa, G: 0xbb, B: 0xcc}, Fill: scene.Color{R: 0x11, G: 0x22, B: 0x33},
		Filled: true, StrokeWidth: 2, Opacity: 1,
	}
	d.Add(r)
	c := &scene.Object{
		Kind: scene.KindEllipse, P1: geom.V(0, 0), P2: geom.V(10, 10),
		Stroke: scene.Color{R: 0xff}, StrokeWidth: 1, Opacity: 1,
	}
	d.Add(c)

	imported, skipped, err := Import(Export(d))
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 {
		t.Errorf("round-trip skipped %d elements", skipped)
	}
	if len(imported.Objects) != 2 {
		t.Fatalf("round-trip objects = %d, want 2", len(imported.Objects))
	}
	got := imported.Objects[0]
	if got.Kind != scene.KindRect || !got.Filled || got.Stroke != r.Stroke || got.Fill != r.Fill {
		t.Errorf("rect round-trip mismatch: %+v", got)
	}
	if got.P1 != r.P1 || got.P2 != r.P2 {
		t.Errorf("rect geometry mismatch: %v %v", got.P1, got.P2)
	}
	if imported.Objects[1].Kind != scene.KindEllipse {
		t.Errorf("ellipse round-trip kind = %s", imported.Objects[1].Kind)
	}
}

func TestImportStyleAttributeAndNamedColors(t *testing.T) {
	src := `<svg xmlns="http://www.w3.org/2000/svg">
	  <rect x="0" y="0" width="4" height="4" style="fill: red; stroke: #00f; stroke-width: 3"/>
	  <circle cx="5" cy="5" r="2" fill="rgb(10, 20, 30)"/>
	  <foreignObject/>
	</svg>`
	doc, skipped, err := Import([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 1 {
		t.Errorf("skipped = %d, want 1 (foreignObject)", skipped)
	}
	r := doc.Objects[0]
	if r.Fill != (scene.Color{R: 0xff}) || r.Stroke != (scene.Color{B: 0xff}) || r.StrokeWidth != 3 {
		t.Errorf("style parsing failed: %+v", r)
	}
	if doc.Objects[1].Fill != (scene.Color{R: 10, G: 20, B: 30}) {
		t.Errorf("rgb() parsing failed: %+v", doc.Objects[1])
	}
}

func TestExportPhase9Features(t *testing.T) {
	d := scene.NewDoc()
	f2 := scene.Color{R: 0xff, G: 0x00, B: 0xff}
	grad := &scene.Object{
		Kind: scene.KindRect, P1: geom.V(0, 0), P2: geom.V(10, 10),
		Stroke: scene.Color{R: 255}, Fill: scene.Color{B: 255}, Fill2: &f2,
		Filled: true, StrokeWidth: 1, Opacity: 1, Shadow: true, Blur: 1,
	}
	d.Add(grad)
	bez := &scene.Object{
		Kind: scene.KindBezier, P1: geom.V(0, 0), C1: geom.V(3, 8), C2: geom.V(7, 8), P2: geom.V(10, 0),
		Stroke: scene.Color{G: 255}, StrokeWidth: 1, Opacity: 1,
	}
	d.Add(bez)
	vw := &scene.Object{
		Kind:   scene.KindPath,
		Points: []geom.Vec{{X: 0, Y: 0}, {X: 2, Y: 1}, {X: 4, Y: 0}},
		Widths: []float64{0.5, 1.5, 2.5},
		Stroke: scene.Color{R: 128}, StrokeWidth: 1, Opacity: 1,
	}
	d.Add(vw)

	out := string(Export(d))
	for _, want := range []string{
		"<linearGradient", `fill="url(#grad`, "<feDropShadow", "<feGaussianBlur",
		`filter="url(#fx`, `d="M 0 0 C 3 8, 7 8, 10 0"`, `stroke-width="0.5"`, `stroke-width="1.5"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("export missing %q", want)
		}
	}
	// Must still be valid XML.
	dec := xml.NewDecoder(strings.NewReader(out))
	for {
		if _, err := dec.Token(); err != nil {
			if err.Error() == "EOF" {
				break
			}
			t.Fatalf("invalid XML: %v", err)
		}
	}
	// Bezier exports must re-import (flattened to a path).
	imported, _, err := Import([]byte(out))
	if err != nil {
		t.Fatal(err)
	}
	foundPath := false
	for _, o := range imported.Objects {
		if o.Kind == scene.KindPath && len(o.Points) > 10 {
			foundPath = true
		}
	}
	if !foundPath {
		t.Error("bezier path did not re-import as flattened path")
	}
}

func TestFlattenPathD(t *testing.T) {
	pts, closed := flattenPathD("M 0 0 L 10 0 L 10 10 Z")
	if !closed || len(pts) != 3 {
		t.Errorf("triangle: closed=%v len=%d", closed, len(pts))
	}
	pts, closed = flattenPathD("M0,0 c 10,0 10,10 20,10")
	if closed || len(pts) < 10 {
		t.Errorf("relative cubic: closed=%v len=%d", closed, len(pts))
	}
	if end := pts[len(pts)-1]; end.Dist(geom.V(20, 10)) > 0.01 {
		t.Errorf("cubic endpoint = %v, want (20,10)", end)
	}
	pts, _ = flattenPathD("M 0 0 H 10 V 5 h -2 v -1")
	if end := pts[len(pts)-1]; end.Dist(geom.V(8, 4)) > 0.01 {
		t.Errorf("H/V endpoint = %v, want (8,4)", end)
	}
	if p, _ := flattenPathD("M 0 0 A 5 5 0 0 1 10 10"); p != nil {
		t.Error("arcs should be rejected, not mangled")
	}
}
