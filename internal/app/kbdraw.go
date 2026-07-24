package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/anishhs-gh/canvux/internal/geom"
	"github.com/anishhs-gh/canvux/internal/render"
	"github.com/anishhs-gh/canvux/internal/scene"
)

// Keyboard-only drawing (accessibility): a virtual cursor is moved with the
// arrow keys and points are set with enter/space, so shapes can be drawn
// without a mouse.

func (m *Model) toggleKbDraw() tea.Cmd {
	m.kbDraw = !m.kbDraw
	if m.kbDraw {
		m.kbCursor = m.maybeSnap(m.doc.Camera.Center)
		m.setStatus(statusInfo, "keyboard draw: arrows move · enter set point · esc exit")
	} else {
		m.draft = nil
		m.polyPts = nil
		m.setStatus(statusInfo, "keyboard draw off")
	}
	return nil
}

// kbStep is the cursor move distance per arrow press (grid step when snapping,
// else a few world units scaled to the current zoom).
func (m *Model) kbStep(fine bool) float64 {
	if m.snap {
		return render.GridStep(m.doc.Camera.Zoom)
	}
	step := 6 / m.doc.Camera.Zoom
	if fine {
		step = 1 / m.doc.Camera.Zoom
	}
	return step
}

// handleKbDrawKey handles the keys special to keyboard-draw mode. It returns
// (cmd, true) when it consumed the key, else (nil, false) to fall through to
// the normal keymap.
func (m *Model) handleKbDrawKey(key string, msg tea.KeyMsg) (tea.Cmd, bool) {
	dir, isArrow := kbDir[key]
	switch {
	case key == "esc":
		if m.draft != nil || m.polyPts != nil {
			m.draft, m.polyPts = nil, nil
			m.setStatus(statusInfo, "keyboard draw: cleared")
		} else {
			return m.toggleKbDraw(), true
		}
		return nil, true
	case isArrow:
		fine := len(key) > 5 && key[:6] == "shift+"
		m.kbCursor = m.kbCursor.Add(dir.Mul(m.kbStep(fine)))
		// Keep the cursor in view by nudging the camera if it drifts off.
		m.followKbCursor()
		m.updateKbDraft()
		return nil, true
	case key == "enter" || key == " " || key == "space":
		return m.kbPlacePoint(), true
	}
	return nil, false
}

var kbDir = map[string]geom.Vec{
	"up": {X: 0, Y: -1}, "down": {X: 0, Y: 1}, "left": {X: -1, Y: 0}, "right": {X: 1, Y: 0},
	"shift+up": {X: 0, Y: -1}, "shift+down": {X: 0, Y: 1},
	"shift+left": {X: -1, Y: 0}, "shift+right": {X: 1, Y: 0},
}

// kbPlacePoint sets a point at the virtual cursor according to the active tool.
func (m *Model) kbPlacePoint() tea.Cmd {
	c := m.maybeSnap(m.kbCursor)
	switch m.tool {
	case ToolLine, ToolRect, ToolEllipse, ToolArrow, ToolBezier:
		if m.draft == nil {
			m.startKbDraft(c)
			m.setStatus(statusInfo, "start set — move and press enter to finish")
		} else {
			m.draft.P2 = c
			if m.draft.Kind == scene.KindBezier {
				d := m.draft.P2.Sub(m.draft.P1)
				m.draft.C1 = m.draft.P1.Add(d.Mul(1.0 / 3.0))
				m.draft.C2 = m.draft.P1.Add(d.Mul(2.0 / 3.0))
			}
			m.commitDraft()
		}
	case ToolPolygon:
		if len(m.polyPts) >= 3 && c.Dist(m.polyPts[0]) <= m.alignTolerance() {
			m.finishPolygon()
		} else {
			m.polyPts = append(m.polyPts, c)
			m.setStatus(statusInfo, "polygon: %d points (enter near start to close)", len(m.polyPts))
		}
	case ToolText:
		o := m.newObject(scene.KindText)
		o.P1 = c
		m.textObj, m.editText, m.textCaret = o, nil, 0
	default:
		m.setStatus(statusInfo, "keyboard draw: switch to a shape/line/text tool")
	}
	return nil
}

func (m *Model) startKbDraft(c geom.Vec) {
	kinds := map[Tool]scene.Kind{
		ToolLine: scene.KindLine, ToolRect: scene.KindRect,
		ToolEllipse: scene.KindEllipse, ToolArrow: scene.KindArrow, ToolBezier: scene.KindBezier,
	}
	m.draft = m.newObject(kinds[m.tool])
	m.draft.P1, m.draft.P2 = c, c
	m.draft.C1, m.draft.C2 = c, c
}

// updateKbDraft keeps the in-progress draft's endpoint following the cursor.
func (m *Model) updateKbDraft() {
	if m.draft == nil {
		return
	}
	c := m.maybeSnap(m.kbCursor)
	m.draft.P2 = c
	if m.draft.Kind == scene.KindBezier {
		d := m.draft.P2.Sub(m.draft.P1)
		m.draft.C1 = m.draft.P1.Add(d.Mul(1.0 / 3.0))
		m.draft.C2 = m.draft.P1.Add(d.Mul(2.0 / 3.0))
	}
}

// followKbCursor pans the camera so the virtual cursor stays on screen.
func (m *Model) followKbCursor() {
	wr := m.view().WorldRect()
	margin := wr.W() * 0.1
	c := m.kbCursor
	if c.X < wr.Min.X+margin || c.X > wr.Max.X-margin ||
		c.Y < wr.Min.Y+margin || c.Y > wr.Max.Y-margin {
		m.doc.Camera.Center = c
	}
}
