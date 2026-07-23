package app

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/anishhs-gh/canvux/internal/geom"
	"github.com/anishhs-gh/canvux/internal/render"
	"github.com/anishhs-gh/canvux/internal/scene"
)

// View renders one full frame.
func (m *Model) View() string {
	if m.w < 20 || m.h < 6 {
		return "Terminal too small for Canvux — resize to at least 20x6."
	}
	if m.present {
		return m.viewPresent()
	}
	t := m.theme
	g := render.NewCellGrid(m.w, m.h, t.CanvasBG)
	v := m.view()
	pb := render.NewPixelBuf(v.W, v.H)

	if m.showGrid {
		render.DrawGrid(pb, v, t.GridDot, t.Axis)
	}
	for _, o := range m.doc.VisibleObjects() {
		render.DrawObject(pb, v, o)
	}
	if m.draft != nil {
		render.DrawObject(pb, v, m.draft)
	}
	m.drawPolygonPreview(pb, v)
	m.drawSelection(pb, v)
	m.drawMarquee(pb, v)

	g.Composite(pb, m.mode, 1, t.CanvasBG)
	m.drawTexts(g, v)
	m.drawPeers(g, v)
	m.drawToolbar(g)
	m.drawStatus(g)
	m.drawOverlay(g)
	return g.ANSI()
}

// drawPeers renders remote collaborators' live cursors.
func (m *Model) drawPeers(g *render.CellGrid, v render.View) {
	if len(m.peers) == 0 {
		return
	}
	sx, sy := m.mode.PixelScale()
	for _, p := range m.peers {
		px := v.ToPixel(p.pos)
		cx, cy := int(px.X)/sx, int(px.Y)/sy+1
		under := g.Get(cx, cy)
		g.Set(cx, cy, render.Cell{Ch: '✛', Fg: p.color, Bg: under.Bg})
		for i, r := range p.name {
			u := g.Get(cx+2+i, cy)
			g.Set(cx+2+i, cy, render.Cell{Ch: r, Fg: p.color, Bg: u.Bg})
		}
	}
}

// hairline returns a world-space stroke width that rasterizes to ~1px.
func (m *Model) hairline() float64 { return 0.001 }

func (m *Model) drawPolygonPreview(pb *render.PixelBuf, v render.View) {
	if len(m.polyPts) == 0 {
		return
	}
	pts := append(append([]geom.Vec(nil), m.polyPts...), m.maybeSnap(m.mouseWorld))
	o := &scene.Object{
		Kind: scene.KindPath, Points: pts,
		Stroke: Palette[m.strokeIdx], StrokeWidth: m.strokeWidth, Opacity: 0.8,
	}
	render.DrawObject(pb, v, o)
	// dashed closing edge back to the first vertex
	if len(pts) > 2 {
		closing := &scene.Object{
			Kind: scene.KindLine, P1: pts[len(pts)-1], P2: pts[0],
			Stroke: m.theme.BarDim, StrokeWidth: m.hairline(), Opacity: 0.8, Dashed: true,
		}
		render.DrawObject(pb, v, closing)
	}
}

func (m *Model) drawSelection(pb *render.PixelBuf, v render.View) {
	objs := m.selection()
	if len(objs) == 0 {
		return
	}
	for _, o := range objs {
		b := o.Bounds().Expand(1 / v.Zoom)
		box := &scene.Object{
			Kind: scene.KindRect, P1: b.Min, P2: b.Max,
			Stroke: m.theme.Selection, StrokeWidth: m.hairline(), Opacity: 1, Dashed: true,
		}
		render.DrawObject(pb, v, box)
	}
	// Resize handles for single selection.
	if len(objs) == 1 {
		b := objs[0].Bounds()
		hs := 2.5 / v.Zoom
		for _, c := range b.Corners() {
			h := &scene.Object{
				Kind: scene.KindRect,
				P1:   c.Sub(geom.V(hs, hs)), P2: c.Add(geom.V(hs, hs)),
				Stroke: m.theme.Handle, Fill: m.theme.Handle, Filled: true,
				StrokeWidth: m.hairline(), Opacity: 1,
			}
			render.DrawObject(pb, v, h)
		}
		// Bézier control points: guide lines + round handles.
		if o := objs[0]; o.Kind == scene.KindBezier {
			for _, g := range [][2]geom.Vec{{o.P1, o.C1}, {o.P2, o.C2}} {
				guide := &scene.Object{
					Kind: scene.KindLine, P1: g[0], P2: g[1],
					Stroke: m.theme.BarDim, StrokeWidth: m.hairline(), Opacity: 0.8, Dashed: true,
				}
				render.DrawObject(pb, v, guide)
				knob := &scene.Object{
					Kind: scene.KindEllipse,
					P1:   g[1].Sub(geom.V(hs, hs)), P2: g[1].Add(geom.V(hs, hs)),
					Stroke: m.theme.Accent, Fill: m.theme.Accent, Filled: true,
					StrokeWidth: m.hairline(), Opacity: 1,
				}
				render.DrawObject(pb, v, knob)
			}
		}
	}
}

func (m *Model) drawMarquee(pb *render.PixelBuf, v render.View) {
	if m.drag.kind != dragMarquee {
		return
	}
	r := geom.R(m.drag.start, m.mouseWorld)
	o := &scene.Object{
		Kind: scene.KindRect, P1: r.Min, P2: r.Max,
		Stroke: m.theme.Marquee, StrokeWidth: m.hairline(), Opacity: 1, Dashed: true,
	}
	render.DrawObject(pb, v, o)
}

// drawTexts composits text objects at the cell layer (terminal-native text).
func (m *Model) drawTexts(g *render.CellGrid, v render.View) {
	sx, sy := m.mode.PixelScale()
	place := func(o *scene.Object, editing bool) {
		p := v.ToPixel(o.P1)
		cx := int(p.X) / sx
		cy := int(p.Y)/sy + 1
		for i, r := range o.Text {
			under := g.Get(cx+i, cy)
			bg := under.Bg
			if m.sel[o.ID] {
				bg = m.theme.OverlaySel
			}
			g.Set(cx+i, cy, render.Cell{Ch: r, Fg: o.Stroke, Bg: bg})
		}
		if editing {
			n := len([]rune(o.Text))
			g.Set(cx+n, cy, render.Cell{Ch: ' ', Fg: m.theme.AccentText, Bg: m.theme.Accent})
		}
	}
	for _, o := range m.doc.VisibleObjects() {
		if o.Kind == scene.KindText {
			place(o, false)
		}
	}
	if m.textObj != nil {
		place(m.textObj, true)
	}
}

// --- toolbar ---

type toolButton struct {
	label string
	key   string
	tool  Tool
}

var toolButtons = []toolButton{
	{"Sel", "v", ToolSelect}, {"Pan", "␣", ToolPan}, {"Brush", "b", ToolBrush},
	{"Line", "l", ToolLine}, {"Rect", "r", ToolRect}, {"Ellip", "e", ToolEllipse},
	{"Arrow", "a", ToolArrow}, {"Poly", "p", ToolPolygon}, {"Curve", "n", ToolBezier},
	{"Text", "t", ToolText}, {"Erase", "x", ToolEraser},
}

func (m *Model) drawToolbar(g *render.CellGrid) {
	t := m.theme
	g.FillRow(0, t.BarBG)
	m.toolbarHits = m.toolbarHits[:0]
	x := 0
	put := func(s string, fg, bg scene.Color, action func(*Model) tea.Cmd) {
		start := x
		g.SetString(x, 0, s, fg, bg)
		x += len([]rune(s))
		if action != nil {
			m.toolbarHits = append(m.toolbarHits, hitRegion{start, x, action})
		}
	}
	put(" ◆ Canvux ", t.Accent, t.BarBG, nil)
	put("│", t.BarDim, t.BarBG, nil)
	for _, b := range toolButtons {
		b := b
		fg, bg := t.BarDim, t.BarBG
		if m.tool == b.tool {
			fg, bg = t.AccentText, t.Accent
		}
		put(fmt.Sprintf(" %s %s ", b.key, b.label), fg, bg,
			func(m *Model) tea.Cmd { m.setTool(b.tool); return nil })
	}
	put("│", t.BarDim, t.BarBG, nil)
	// Style cluster: stroke swatch, fill swatch, width, dash.
	put(" ", t.BarFG, t.BarBG, nil)
	put("▣", Palette[m.strokeIdx], t.BarBG,
		func(m *Model) tea.Cmd { m.overlay = ovColorStroke; m.colorSel = m.strokeIdx; return nil })
	put(" ", t.BarFG, t.BarBG, nil)
	fill := "□"
	if m.filled {
		fill = "■"
	}
	put(fill, Palette[m.fillIdx], t.BarBG,
		func(m *Model) tea.Cmd { m.overlay = ovColorFill; m.colorSel = m.fillIdx; return nil })
	dash := "—"
	if m.dashed {
		dash = "┄"
	}
	put(" "+dash, t.BarFG, t.BarBG, func(m *Model) tea.Cmd { return m.cmdToggleDash() })
	put(fmt.Sprintf(" %.1f ", m.strokeWidth), t.BarFG, t.BarBG,
		func(m *Model) tea.Cmd { return m.cmdStrokeWidth(0.5) })

	// Right side: palette + help buttons.
	right := " ⌘ cmds  ? help "
	rx := m.w - len([]rune(right))
	if rx > x {
		g.SetString(rx, 0, right, t.BarDim, t.BarBG)
		m.toolbarHits = append(m.toolbarHits, hitRegion{rx, rx + 8, func(m *Model) tea.Cmd {
			m.overlay = ovPalette
			m.palQuery = ""
			m.palSel = 0
			return nil
		}})
		m.toolbarHits = append(m.toolbarHits, hitRegion{rx + 8, rx + 16, func(m *Model) tea.Cmd {
			m.overlay = ovHelp
			m.helpTop = 0
			return nil
		}})
	}
}

// --- status bar ---

func (m *Model) drawStatus(g *render.CellGrid) {
	t := m.theme
	y := m.h - 1
	g.FillRow(y, t.BarBG)

	if m.prompt != nil {
		p := m.prompt
		label := " " + p.label + ": "
		g.SetString(0, y, label, t.Accent, t.BarBG)
		if !p.yesNo {
			g.SetString(len([]rune(label)), y, p.value, t.BarFG, t.BarBG)
			cx := len([]rune(label)) + len([]rune(p.value))
			g.Set(cx, y, render.Cell{Ch: ' ', Fg: t.AccentText, Bg: t.Accent})
		}
		return
	}

	// Left: file, dirty, status message.
	name := m.path
	if name == "" {
		name = "untitled"
	}
	dirty := ""
	if m.dirty {
		dirty = "●"
	}
	left := fmt.Sprintf(" %s%s", name, dirty)
	g.SetString(0, y, left, t.BarFG, t.BarBG)
	lx := len([]rune(left))

	// Right: coordinates, zoom, counts, layer, flags (drawn first so the
	// left-side message can be truncated to fit).
	flags := ""
	if m.showGrid {
		flags += "grid "
	}
	if m.snap {
		flags += "snap "
	}
	layer := m.doc.Layers[m.currentLayer()].Name
	peers := ""
	if m.collab != nil {
		peers = fmt.Sprintf("◉ %d peers · ", len(m.peers))
	}
	right := fmt.Sprintf("%s%s· %s · %s · %d obj · %.0f%% · (%.1f, %.1f) ",
		peers, flags, m.mode, layer, len(m.doc.Objects), m.doc.Camera.Zoom*50, m.mouseWorld.X, m.mouseWorld.Y)
	rx := m.w - len([]rune(right))
	if rx > lx {
		g.SetString(rx, y, right, t.BarDim, t.BarBG)
	} else {
		rx = m.w
	}

	msg, fg := "", t.BarDim
	if m.statusMsg != "" && time.Since(m.statusAt) < 6*time.Second {
		msg = m.statusMsg
		switch m.statusLvl {
		case statusOK:
			fg = t.OK
		case statusErr:
			fg = t.Danger
		}
	} else {
		msg = m.toolHint()
	}
	if avail := rx - 2 - (lx + 2); msg != "" && avail > 0 {
		r := []rune(msg)
		if len(r) > avail {
			r = append(r[:maxInt(0, avail-1)], '…')
		}
		g.SetString(lx+2, y, string(r), fg, t.BarBG)
	}
}

func (m *Model) toolHint() string {
	if m.textObj != nil {
		return "type text · enter commit · esc cancel"
	}
	switch m.tool {
	case ToolSelect:
		return "click select · drag move · shift multi · corners resize · del delete"
	case ToolPan:
		return "drag to pan · wheel zoom · space back to previous tool"
	case ToolPolygon:
		if len(m.polyPts) > 0 {
			return fmt.Sprintf("%d points · click to add · enter/double-click close · esc cancel", len(m.polyPts))
		}
		return "click to place vertices · enter closes"
	case ToolBrush:
		if m.varBrush {
			return "drag to draw · stroke speed varies the width"
		}
		return "drag to draw freehand"
	case ToolBezier:
		return "drag to place a curve · then drag its round handles with select"
	case ToolText:
		return "click to place text"
	case ToolEraser:
		return "click or drag over objects to erase"
	default:
		return "drag to draw · shift constrains · right-drag pan · wheel zoom"
	}
}
