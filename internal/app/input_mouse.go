package app

import (
	"math"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/anishhs-gh/canvux/internal/geom"
	"github.com/anishhs-gh/canvux/internal/scene"
)

var lastClick struct {
	at   time.Time
	x, y int
}

func (m *Model) handleMouse(msg tea.MouseMsg) tea.Cmd {
	m.mouseCell.X, m.mouseCell.Y = msg.X, msg.Y
	m.mouseWorld = m.cellToWorld(msg.X, msg.Y)

	// Wheel: zoom at cursor; shift+wheel pans horizontally.
	if msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown {
		if msg.Shift {
			d := 6 / m.doc.Camera.Zoom
			if msg.Button == tea.MouseButtonWheelUp {
				d = -d
			}
			m.doc.Camera.Center.X += d
			return nil
		}
		factor := 1.15
		if msg.Button == tea.MouseButtonWheelDown {
			factor = 1 / factor
		}
		m.zoomAt(m.mouseWorld, factor)
		return nil
	}

	switch msg.Action {
	case tea.MouseActionPress:
		return m.mousePress(msg)
	case tea.MouseActionMotion:
		return m.mouseMotion(msg)
	case tea.MouseActionRelease:
		return m.mouseRelease(msg)
	}
	return nil
}

func (m *Model) zoomAt(world geom.Vec, factor float64) {
	cam := &m.doc.Camera
	old := cam.Zoom
	cam.Zoom = math.Max(0.05, math.Min(400, cam.Zoom*factor))
	if cam.Zoom == old {
		return
	}
	// Keep `world` fixed under the cursor.
	cam.Center = world.Sub(world.Sub(cam.Center).Mul(old / cam.Zoom))
}

func (m *Model) mousePress(msg tea.MouseMsg) tea.Cmd {
	// Any open overlay closes on click.
	if m.overlay != ovNone {
		m.overlay = ovNone
		return nil
	}
	// Toolbar clicks.
	if msg.Y == 0 && msg.Button == tea.MouseButtonLeft {
		for _, hr := range m.toolbarHits {
			if msg.X >= hr.x0 && msg.X < hr.x1 {
				return hr.action(m)
			}
		}
		return nil
	}
	if msg.Y >= m.h-1 || msg.Y == 0 {
		return nil
	}

	// Pan with right or middle button from any tool.
	if msg.Button == tea.MouseButtonRight || msg.Button == tea.MouseButtonMiddle {
		m.beginPan(msg)
		return nil
	}
	if msg.Button != tea.MouseButtonLeft {
		return nil
	}

	dbl := time.Since(lastClick.at) < 400*time.Millisecond &&
		absInt(lastClick.x-msg.X) <= 1 && absInt(lastClick.y-msg.Y) <= 1
	lastClick.at, lastClick.x, lastClick.y = time.Now(), msg.X, msg.Y

	w := m.mouseWorld
	switch m.tool {
	case ToolPan:
		m.beginPan(msg)
	case ToolSelect:
		m.selectPress(w, msg.Shift, dbl)
	case ToolBrush:
		m.draft = m.newObject(scene.KindPath)
		m.draft.Points = []geom.Vec{w}
		if m.varBrush {
			m.draft.Widths = []float64{m.strokeWidth}
		}
		m.drag = dragState{kind: dragDraw, start: w, last: w, button: msg.Button}
	case ToolLine, ToolRect, ToolEllipse, ToolArrow, ToolBezier:
		kinds := map[Tool]scene.Kind{
			ToolLine: scene.KindLine, ToolRect: scene.KindRect,
			ToolEllipse: scene.KindEllipse, ToolArrow: scene.KindArrow,
			ToolBezier: scene.KindBezier,
		}
		p := m.maybeSnap(w)
		m.draft = m.newObject(kinds[m.tool])
		m.draft.P1, m.draft.P2 = p, p
		m.draft.C1, m.draft.C2 = p, p
		m.drag = dragState{kind: dragDraw, start: p, last: p, button: msg.Button}
	case ToolPolygon:
		m.polygonClick(w, dbl)
	case ToolText:
		m.textPress(w)
	case ToolEraser:
		m.drag = dragState{kind: dragErase, start: w, last: w, button: msg.Button}
		m.eraseAt(w)
	}
	return nil
}

func (m *Model) beginPan(msg tea.MouseMsg) {
	sx, sy := m.mode.PixelScale()
	m.drag = dragState{
		kind:       dragPan,
		button:     msg.Button,
		panStartPx: geom.V(float64(msg.X*sx), float64(msg.Y*sy)),
		panCenter:  m.doc.Camera.Center,
	}
}

func (m *Model) selectPress(w geom.Vec, shift, dbl bool) {
	// Resize handles take priority for a single selection.
	if h := m.handleAt(w); h >= 0 {
		objs := m.selection()
		m.drag = dragState{
			kind: dragResize, start: w, last: w, handle: h,
			resizeID: objs[0].ID,
		}
		return
	}
	obj := m.doc.HitTest(w, m.hitTolerance())
	if obj == nil {
		if !shift {
			m.clearSelection()
		}
		m.drag = dragState{kind: dragMarquee, start: w, last: w, addToSel: shift}
		return
	}
	if dbl && obj.Kind == scene.KindText {
		m.beginTextEdit(obj)
		return
	}
	if shift {
		if m.sel[obj.ID] {
			delete(m.sel, obj.ID)
		} else {
			m.sel[obj.ID] = true
		}
		return
	}
	if !m.sel[obj.ID] {
		m.clearSelection()
		m.sel[obj.ID] = true
	}
	m.drag = dragState{kind: dragMove, start: w, last: w}
}

// handleAt returns the handle index under world point w, or -1. Handles only
// exist for a single selection: 0..3 are bounds corners (clockwise from
// top-left); for béziers, 4 and 5 are the C1/C2 control points.
func (m *Model) handleAt(w geom.Vec) int {
	objs := m.selection()
	if len(objs) != 1 {
		return -1
	}
	tol := 4 / m.view().Zoom
	// Control points win over corners so overlapping handles stay editable.
	if objs[0].Kind == scene.KindBezier {
		if w.Dist(objs[0].C1) <= tol {
			return 4
		}
		if w.Dist(objs[0].C2) <= tol {
			return 5
		}
	}
	b := objs[0].Bounds()
	for i, c := range b.Corners() {
		if w.Dist(c) <= tol {
			return i
		}
	}
	return -1
}

func (m *Model) polygonClick(w geom.Vec, dbl bool) {
	p := m.maybeSnap(w)
	if len(m.polyPts) >= 3 {
		closeTol := 4 / m.view().Zoom
		if dbl || p.Dist(m.polyPts[0]) <= closeTol {
			m.finishPolygon()
			return
		}
	}
	m.polyPts = append(m.polyPts, p)
}

func (m *Model) finishPolygon() {
	if len(m.polyPts) >= 3 {
		o := m.newObject(scene.KindPolygon)
		o.Points = append([]geom.Vec(nil), m.polyPts...)
		m.checkpoint("draw polygon")
		m.doc.Add(o)
		m.setStatus(statusOK, "polygon (%d points)", len(o.Points))
	}
	m.polyPts = nil
}

func (m *Model) textPress(w geom.Vec) {
	// Clicking an existing text object edits it in place.
	if obj := m.doc.HitTest(w, m.hitTolerance()); obj != nil && obj.Kind == scene.KindText {
		m.beginTextEdit(obj)
		return
	}
	o := m.newObject(scene.KindText)
	o.P1 = m.maybeSnap(w)
	m.textObj = o
	m.editText = nil
	m.textCaret = 0
}

func (m *Model) beginTextEdit(obj *scene.Object) {
	m.checkpoint("edit text")
	m.editText = obj
	c := obj.Clone()
	m.textObj = c
	m.textCaret = len([]rune(c.Text))
	m.doc.Remove(obj.ID) // temporarily lift out; re-added on commit
}

func (m *Model) eraseAt(w geom.Vec) {
	if obj := m.doc.HitTest(w, m.hitTolerance()); obj != nil {
		if !m.drag.pushed {
			m.checkpoint("erase")
			m.drag.pushed = true
		}
		m.doc.Remove(obj.ID)
		delete(m.sel, obj.ID)
	}
}

func (m *Model) mouseMotion(msg tea.MouseMsg) tea.Cmd {
	w := m.mouseWorld
	switch m.drag.kind {
	case dragPan:
		sx, sy := m.mode.PixelScale()
		cur := geom.V(float64(msg.X*sx), float64(msg.Y*sy))
		d := cur.Sub(m.drag.panStartPx).Mul(1 / m.view().Zoom)
		m.doc.Camera.Center = m.drag.panCenter.Sub(d)
	case dragDraw:
		if m.draft == nil {
			break
		}
		if m.draft.Kind == scene.KindPath {
			last := m.draft.Points[len(m.draft.Points)-1]
			if w.Dist(last) > 0.5/m.view().Zoom {
				m.draft.Points = append(m.draft.Points, w)
				if m.draft.Widths != nil {
					// Speed-sensitive width: fast strokes thin out, slow
					// strokes thicken — simulated brush pressure.
					pxDist := w.Dist(m.drag.last) * m.view().Zoom
					f := math.Max(0.35, math.Min(1.6, 1.7-pxDist*0.10))
					prev := m.draft.Widths[len(m.draft.Widths)-1]
					m.draft.Widths = append(m.draft.Widths, prev*0.6+m.strokeWidth*f*0.4)
				}
			}
		} else {
			p := m.constrained(m.maybeSnap(w), msg.Shift)
			m.alignGuides = nil
			if !msg.Shift {
				var guides []guideLine
				p, guides = m.snapPointToObjects(p, nil, m.alignTolerance())
				m.alignGuides = guides
			}
			m.draft.P2 = p
			if m.draft.Kind == scene.KindBezier {
				// Start as a straight curve; handles are edited afterwards.
				d := m.draft.P2.Sub(m.draft.P1)
				m.draft.C1 = m.draft.P1.Add(d.Mul(1.0 / 3.0))
				m.draft.C2 = m.draft.P1.Add(d.Mul(2.0 / 3.0))
			}
		}
		m.drag.last = w
	case dragMove:
		cur, last := m.maybeSnap(w), m.maybeSnap(m.drag.last)
		d := cur.Sub(last)
		if d.X != 0 || d.Y != 0 {
			if !m.drag.pushed {
				m.checkpoint("move")
				m.drag.pushed = true
			}
			for _, o := range m.selection() {
				o.Translate(d)
			}
			m.dirty = true
		}
		// Smart guides: pull the selection onto nearby object edges/centers.
		m.alignGuides = nil
		if !msg.Shift {
			if b, ok := m.selectionBounds(); ok {
				res := m.alignSnap(b, m.sel, m.alignTolerance())
				if res.dx != 0 || res.dy != 0 {
					adj := geom.V(res.dx, res.dy)
					for _, o := range m.selection() {
						o.Translate(adj)
					}
				}
				m.alignGuides = res.guides
			}
		}
		m.drag.last = w
	case dragResize:
		m.applyResize(w)
	case dragMarquee, dragErase:
		if m.drag.kind == dragErase {
			m.eraseAt(w)
		}
		m.drag.last = w
	}
	return nil
}

// constrained squares up rect/ellipse and snaps lines to 45° when shift held.
func (m *Model) constrained(p geom.Vec, shift bool) geom.Vec {
	if !shift || m.draft == nil {
		return p
	}
	d := p.Sub(m.draft.P1)
	switch m.draft.Kind {
	case scene.KindRect, scene.KindEllipse:
		s := math.Max(math.Abs(d.X), math.Abs(d.Y))
		return m.draft.P1.Add(geom.V(math.Copysign(s, d.X), math.Copysign(s, d.Y)))
	case scene.KindLine, scene.KindArrow, scene.KindBezier:
		ang := math.Round(math.Atan2(d.Y, d.X)/(math.Pi/4)) * (math.Pi / 4)
		return m.draft.P1.Add(geom.V(math.Cos(ang), math.Sin(ang)).Mul(d.Len()))
	}
	return p
}

func (m *Model) applyResize(w geom.Vec) {
	obj := m.doc.Get(m.drag.resizeID)
	if obj == nil {
		return
	}
	if !m.drag.pushed {
		label := "resize"
		if m.drag.handle >= 4 {
			label = "curve"
		}
		m.checkpoint(label)
		m.drag.pushed = true
		// re-fetch: checkpoint clones the doc into history but obj stays live
	}
	// Bézier control-point handles reposition C1/C2 directly.
	if m.drag.handle >= 4 && obj.Kind == scene.KindBezier {
		p := m.maybeSnap(w)
		if m.drag.handle == 4 {
			obj.C1 = p
		} else {
			obj.C2 = p
		}
		m.dirty = true
		return
	}
	b := obj.Bounds()
	corners := b.Corners()
	pivot := corners[(m.drag.handle+2)%4] // opposite corner
	cur := m.maybeSnap(w)
	start := corners[m.drag.handle]
	dx0, dy0 := start.X-pivot.X, start.Y-pivot.Y
	if math.Abs(dx0) < 1e-9 || math.Abs(dy0) < 1e-9 {
		return
	}
	sx := (cur.X - pivot.X) / dx0
	sy := (cur.Y - pivot.Y) / dy0
	if math.Abs(sx) < 0.01 || math.Abs(sy) < 0.01 {
		return
	}
	obj.ScaleAround(pivot, sx, sy)
	m.dirty = true
}

func (m *Model) mouseRelease(msg tea.MouseMsg) tea.Cmd {
	switch m.drag.kind {
	case dragDraw:
		m.commitDraft()
	case dragMarquee:
		r := geom.R(m.drag.start, m.mouseWorld)
		if !m.drag.addToSel {
			m.clearSelection()
		}
		n := 0
		for _, o := range m.doc.VisibleObjects() {
			if m.doc.Layers[o.Layer].Locked {
				continue
			}
			if o.Bounds().Intersects(r) {
				m.sel[o.ID] = true
				n++
			}
		}
		if n > 0 {
			m.setStatus(statusInfo, "%d selected", len(m.sel))
		}
	}
	m.drag = dragState{}
	m.alignGuides = nil
	return nil
}

func (m *Model) commitDraft() {
	d := m.draft
	m.draft = nil
	if d == nil {
		return
	}
	// Reject degenerate shapes (accidental clicks).
	switch d.Kind {
	case scene.KindPath:
		if len(d.Points) < 2 {
			return
		}
		// Smooth & decimate freehand strokes so they're clean and compact.
		// Variable-width strokes carry a per-point width, so resample that too.
		tol := 0.6 / m.view().Zoom
		if len(d.Widths) == len(d.Points) {
			d.Points, d.Widths = smoothVarStroke(d.Points, d.Widths, tol)
		} else {
			d.Points = geom.SmoothStroke(d.Points, tol)
		}
	default:
		if d.P1.Dist(d.P2) < 0.3/m.view().Zoom {
			return
		}
	}
	m.checkpoint("draw " + string(d.Kind))
	m.doc.Add(d)
	m.clearSelection()
	m.sel[d.ID] = true
}

// smoothVarStroke decimates a variable-width stroke by distance and resamples
// widths onto the kept points (nearest original point), then leaves the point
// count as-is (no Chaikin, to keep width/point correspondence simple).
func smoothVarStroke(pts []geom.Vec, widths []float64, tol float64) ([]geom.Vec, []float64) {
	if len(pts) < 3 {
		return pts, widths
	}
	// Greedy distance decimation preserving indices so widths stay aligned.
	keptP := []geom.Vec{pts[0]}
	keptW := []float64{widths[0]}
	last := pts[0]
	for i := 1; i < len(pts)-1; i++ {
		if pts[i].Dist(last) >= tol {
			keptP = append(keptP, pts[i])
			keptW = append(keptW, widths[i])
			last = pts[i]
		}
	}
	keptP = append(keptP, pts[len(pts)-1])
	keptW = append(keptW, widths[len(widths)-1])
	return keptP, keptW
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
