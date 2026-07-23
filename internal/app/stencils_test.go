package app

import (
	"testing"

	"github.com/anishhs-gh/canvux/internal/scene"
)

// Every stencil must build valid, drawable, hit-testable objects.
func TestStencilsAreWellFormed(t *testing.T) {
	valid := map[scene.Kind]bool{
		scene.KindLine: true, scene.KindRect: true, scene.KindEllipse: true,
		scene.KindPolygon: true, scene.KindPath: true, scene.KindArrow: true,
		scene.KindText: true, scene.KindBezier: true,
	}
	seen := map[string]bool{}
	for _, s := range Stencils {
		if s.Name == "" || s.Cat == "" {
			t.Errorf("stencil %+v missing name or category", s)
		}
		if seen[s.Name] {
			t.Errorf("duplicate stencil name %q", s.Name)
		}
		seen[s.Name] = true
		objs := s.Build()
		if len(objs) == 0 {
			t.Errorf("%s: builds no objects", s.Name)
			continue
		}
		for i, o := range objs {
			if !valid[o.Kind] {
				t.Errorf("%s[%d]: invalid kind %q", s.Name, i, o.Kind)
			}
			if o.Opacity <= 0 || o.StrokeWidth <= 0 {
				t.Errorf("%s[%d]: zero opacity or stroke width", s.Name, i)
			}
			if o.Kind == scene.KindText && o.Text == "" {
				t.Errorf("%s[%d]: empty text object", s.Name, i)
			}
			if o.Kind != scene.KindText {
				b := o.Bounds()
				if b.W() <= 0 && b.H() <= 0 {
					t.Errorf("%s[%d]: degenerate bounds %+v", s.Name, i, b)
				}
			}
		}
	}
	if len(Stencils) < 15 {
		t.Errorf("stencil library has only %d entries", len(Stencils))
	}
}
