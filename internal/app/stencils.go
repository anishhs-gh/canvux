package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/anishhs-gh/canvux/internal/geom"
	"github.com/anishhs-gh/canvux/internal/scene"
)

// Stencil is a prefab group of objects (Phase 10 diagramming library).
type Stencil struct {
	Name  string
	Cat   string
	Build func() []*scene.Object
}

// --- tiny builders (world units; ~1 unit per terminal column) ---

func stObj(kind scene.Kind, stroke scene.Color) *scene.Object {
	return &scene.Object{Kind: kind, Stroke: stroke, StrokeWidth: 1, Opacity: 1}
}

func stRect(x1, y1, x2, y2 float64, stroke, fill scene.Color, filled bool) *scene.Object {
	o := stObj(scene.KindRect, stroke)
	o.P1, o.P2 = geom.V(x1, y1), geom.V(x2, y2)
	o.Fill, o.Filled = fill, filled
	return o
}

func stEllipse(x1, y1, x2, y2 float64, stroke, fill scene.Color, filled bool) *scene.Object {
	o := stObj(scene.KindEllipse, stroke)
	o.P1, o.P2 = geom.V(x1, y1), geom.V(x2, y2)
	o.Fill, o.Filled = fill, filled
	return o
}

func stLine(x1, y1, x2, y2 float64, stroke scene.Color) *scene.Object {
	o := stObj(scene.KindLine, stroke)
	o.P1, o.P2 = geom.V(x1, y1), geom.V(x2, y2)
	return o
}

func stArrow(x1, y1, x2, y2 float64, stroke scene.Color) *scene.Object {
	o := stObj(scene.KindArrow, stroke)
	o.P1, o.P2 = geom.V(x1, y1), geom.V(x2, y2)
	return o
}

func stPoly(stroke, fill scene.Color, filled bool, xy ...float64) *scene.Object {
	o := stObj(scene.KindPolygon, stroke)
	for i := 0; i+1 < len(xy); i += 2 {
		o.Points = append(o.Points, geom.V(xy[i], xy[i+1]))
	}
	o.Fill, o.Filled = fill, filled
	return o
}

func stText(x, y float64, s string, c scene.Color) *scene.Object {
	o := stObj(scene.KindText, c)
	o.P1 = geom.V(x, y)
	o.Text = s
	return o
}

// stencil palette shorthands
var (
	stBlue   = Palette[1]
	stCyan   = Palette[2]
	stGreen  = Palette[3]
	stAmber  = Palette[4]
	stRed    = Palette[6]
	stPurple = Palette[7]
	stFg     = Palette[0]
	stSlate  = Palette[9]

	stBlueBG  = scene.Color{R: 0x1a, G: 0x23, B: 0x40}
	stGreenBG = scene.Color{R: 0x1c, G: 0x2a, B: 0x1a}
	stAmberBG = scene.Color{R: 0x3a, G: 0x2f, B: 0x14}
	stDarkBG  = scene.Color{R: 0x20, G: 0x24, B: 0x30}
)

// Stencils is the built-in diagram library, shown by the Insert overlay (i).
var Stencils = []Stencil{
	// Flowchart
	{"Process", "flowchart", func() []*scene.Object {
		return []*scene.Object{
			stRect(0, 0, 20, 8, stBlue, stBlueBG, true),
			stText(7, 3, "step", stFg),
		}
	}},
	{"Decision", "flowchart", func() []*scene.Object {
		return []*scene.Object{
			stPoly(stAmber, stAmberBG, true, 11, 0, 22, 6, 11, 12, 0, 6),
			stText(9, 5, "if?", stFg),
		}
	}},
	{"Terminator", "flowchart", func() []*scene.Object {
		return []*scene.Object{
			stEllipse(0, 0, 18, 7, stGreen, stGreenBG, true),
			stText(6, 2.5, "start", stFg),
		}
	}},
	{"Input / Output", "flowchart", func() []*scene.Object {
		return []*scene.Object{
			stPoly(stCyan, stDarkBG, true, 4, 0, 24, 0, 20, 8, 0, 8),
			stText(8, 3, "data", stFg),
		}
	}},
	{"Database", "flowchart", func() []*scene.Object {
		return []*scene.Object{
			stRect(0, 3, 16, 13, stPurple, stDarkBG, true),
			stEllipse(0, 0, 16, 6, stPurple, stDarkBG, true),
			stText(5, 7, "db", stFg),
		}
	}},
	{"Document", "flowchart", func() []*scene.Object {
		return []*scene.Object{
			stPoly(stFg, stDarkBG, true, 0, 0, 20, 0, 20, 9, 15, 7.5, 10, 9, 5, 7.5, 0, 9),
			stText(6, 3, "doc", stFg),
		}
	}},
	// UML
	{"Class", "uml", func() []*scene.Object {
		return []*scene.Object{
			stRect(0, 0, 22, 14, stBlue, stDarkBG, true),
			stLine(0, 4, 22, 4, stBlue),
			stLine(0, 9, 22, 9, stBlue),
			stText(6, 1, "Class", stFg),
			stText(2, 5.5, "+ field", stSlate),
			stText(2, 10.5, "+ method()", stSlate),
		}
	}},
	{"Actor", "uml", func() []*scene.Object {
		return []*scene.Object{
			stEllipse(3, 0, 9, 5, stFg, stDarkBG, false),
			stLine(6, 5, 6, 12, stFg),   // body
			stLine(1, 7, 11, 7, stFg),   // arms
			stLine(6, 12, 2, 17, stFg),  // left leg
			stLine(6, 12, 10, 17, stFg), // right leg
			stText(2, 18, "actor", stSlate),
		}
	}},
	{"Use Case", "uml", func() []*scene.Object {
		return []*scene.Object{
			stEllipse(0, 0, 22, 9, stCyan, stDarkBG, true),
			stText(6, 3.5, "use case", stFg),
		}
	}},
	{"Note", "uml", func() []*scene.Object {
		return []*scene.Object{
			stPoly(stAmber, stAmberBG, true, 0, 0, 15, 0, 18, 3, 18, 11, 0, 11),
			stLine(15, 0, 15, 3, stAmber),
			stLine(15, 3, 18, 3, stAmber),
			stText(2, 5, "note...", stFg),
		}
	}},
	// ER
	{"Entity", "er", func() []*scene.Object {
		return []*scene.Object{
			stRect(0, 0, 18, 8, stGreen, stGreenBG, true),
			stText(5, 3, "entity", stFg),
		}
	}},
	{"Relationship", "er", func() []*scene.Object {
		return []*scene.Object{
			stPoly(stRed, stDarkBG, true, 11, 0, 22, 6, 11, 12, 0, 6),
			stText(8, 5, "has", stFg),
		}
	}},
	{"Attribute", "er", func() []*scene.Object {
		return []*scene.Object{
			stEllipse(0, 0, 16, 7, stPurple, stDarkBG, true),
			stText(4, 2.5, "attr", stFg),
		}
	}},
	// Architecture
	{"Server", "arch", func() []*scene.Object {
		return []*scene.Object{
			stRect(0, 0, 14, 18, stBlue, stDarkBG, true),
			stLine(0, 5, 14, 5, stBlue),
			stLine(0, 10, 14, 10, stBlue),
			stEllipse(10, 1.5, 12, 3.5, stGreen, stGreen, true),
			stEllipse(10, 6.5, 12, 8.5, stGreen, stGreen, true),
			stText(2, 13, "srv", stFg),
		}
	}},
	{"Cloud", "arch", func() []*scene.Object {
		return []*scene.Object{
			stEllipse(0, 4, 12, 12, stCyan, stDarkBG, true),
			stEllipse(6, 0, 20, 10, stCyan, stDarkBG, true),
			stEllipse(12, 4, 26, 12, stCyan, stDarkBG, true),
			stText(9, 6, "cloud", stFg),
		}
	}},
	{"Queue", "arch", func() []*scene.Object {
		return []*scene.Object{
			stRect(0, 0, 20, 6, stAmber, stDarkBG, true),
			stLine(5, 0, 5, 6, stAmber),
			stLine(10, 0, 10, 6, stAmber),
			stLine(15, 0, 15, 6, stAmber),
			stArrow(21, 3, 27, 3, stAmber),
		}
	}},
	{"User", "arch", func() []*scene.Object {
		return []*scene.Object{
			stEllipse(3, 0, 9, 6, stFg, stDarkBG, true),
			stPoly(stFg, stDarkBG, true, 0, 13, 1, 9, 4, 7, 8, 7, 11, 9, 12, 13),
			stText(1, 14, "user", stSlate),
		}
	}},
	// Misc
	{"Sticky Note", "misc", func() []*scene.Object {
		body := stPoly(stAmber, stAmberBG, true, 0, 0, 16, 0, 16, 9, 13, 12, 0, 12)
		body.Shadow = true
		fold := stPoly(stAmber, scene.Color{R: 0x55, G: 0x45, B: 0x1e}, true, 13, 12, 13, 9, 16, 9)
		return []*scene.Object{body, fold, stText(2, 4, "todo...", stFg)}
	}},
	{"Table 3x3", "misc", func() []*scene.Object {
		return []*scene.Object{
			stRect(0, 0, 30, 12, stFg, stDarkBG, true),
			stLine(0, 4, 30, 4, stFg),
			stLine(0, 8, 30, 8, stFg),
			stLine(10, 0, 10, 12, stFg),
			stLine(20, 0, 20, 12, stFg),
			stText(2, 1, "col a", stSlate),
			stText(12, 1, "col b", stSlate),
			stText(22, 1, "col c", stSlate),
		}
	}},
	{"Mind Map", "misc", func() []*scene.Object {
		objs := []*scene.Object{
			stEllipse(18, 10, 34, 18, stBlue, stBlueBG, true),
			stText(23, 13, "topic", stFg),
		}
		branches := []struct {
			x1, y1, x2, y2 float64
			c              scene.Color
		}{
			{20, 11, 8, 3, stGreen}, {32, 11, 44, 3, stAmber},
			{20, 17, 8, 25, stRed}, {32, 17, 44, 25, stPurple},
		}
		for _, br := range branches {
			objs = append(objs,
				stLine(br.x1, br.y1, br.x2, br.y2, br.c),
				stEllipse(br.x2-6, br.y2-3, br.x2+6, br.y2+3, br.c, stDarkBG, true),
				stText(br.x2-4, br.y2-1, "idea", stFg),
			)
		}
		return objs
	}},
}

// filteredStencils fuzzy-filters by name or category.
func (m *Model) filteredStencils() []Stencil {
	if m.stencilQry == "" {
		return Stencils
	}
	q := strings.ToLower(m.stencilQry)
	var out []Stencil
	for _, s := range Stencils {
		if fuzzyMatch(strings.ToLower(s.Cat+" "+s.Name), q) {
			out = append(out, s)
		}
	}
	return out
}

// insertStencil places a stencil centered on the current view and selects it.
func (m *Model) insertStencil(s Stencil) tea.Cmd {
	objs := s.Build()
	if len(objs) == 0 {
		return nil
	}
	b := objs[0].Bounds()
	for _, o := range objs[1:] {
		b = b.Union(o.Bounds())
	}
	off := m.maybeSnap(m.doc.Camera.Center.Sub(b.Center()))
	m.checkpoint("insert " + s.Name)
	m.clearSelection()
	for _, o := range objs {
		o.Translate(off)
		o.Layer = m.currentLayer()
		m.doc.Add(o)
		m.sel[o.ID] = true
	}
	m.setStatus(statusOK, "inserted %s (%d objects) — drag to place", s.Name, len(objs))
	return nil
}
