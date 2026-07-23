// Package app implements the interactive Canvux terminal editor.
package app

import (
	"fmt"
	"math"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/anishhs-gh/canvux/internal/geom"
	"github.com/anishhs-gh/canvux/internal/history"
	"github.com/anishhs-gh/canvux/internal/render"
	"github.com/anishhs-gh/canvux/internal/scene"
)

// Tool is the active editing tool.
type Tool int

const (
	ToolSelect Tool = iota
	ToolPan
	ToolBrush
	ToolLine
	ToolRect
	ToolEllipse
	ToolArrow
	ToolPolygon
	ToolText
	ToolEraser
)

var toolNames = map[Tool]string{
	ToolSelect: "select", ToolPan: "pan", ToolBrush: "brush", ToolLine: "line",
	ToolRect: "rect", ToolEllipse: "ellipse", ToolArrow: "arrow",
	ToolPolygon: "polygon", ToolText: "text", ToolEraser: "eraser",
}

// dragKind describes the in-flight mouse gesture.
type dragKind int

const (
	dragNone dragKind = iota
	dragDraw
	dragMove
	dragResize
	dragMarquee
	dragPan
	dragErase
)

type dragState struct {
	kind       dragKind
	start      geom.Vec // world point at press
	last       geom.Vec
	button     tea.MouseButton
	handle     int  // resize handle index (corner 0..3)
	pushed     bool // history snapshot already taken for this gesture
	resizeID   uint64
	panStartPx geom.Vec // pixel coords at press for panning
	panCenter  geom.Vec // camera center at press
	addToSel   bool     // shift-marquee adds to selection
}

// overlayKind enumerates modal overlays.
type overlayKind int

const (
	ovNone overlayKind = iota
	ovPalette
	ovHelp
	ovColorStroke
	ovColorFill
	ovLayers
)

// promptState is a single-line input on the status row.
type promptState struct {
	label   string
	value   string
	confirm func(m *Model, value string) tea.Cmd
	// yesNo turns the prompt into a y/n confirmation.
	yesNo bool
	onYes func(m *Model) tea.Cmd
	onNo  func(m *Model) tea.Cmd
}

type statusLevel int

const (
	statusInfo statusLevel = iota
	statusOK
	statusErr
)

// Model is the bubbletea model for the whole editor.
type Model struct {
	doc   *scene.Doc
	path  string
	dirty bool
	hist  history.Stack

	w, h  int
	mode  render.Mode
	theme Theme

	tool        Tool
	prevTool    Tool
	strokeIdx   int
	fillIdx     int
	filled      bool
	strokeWidth float64
	dashed      bool
	opacity     float64

	sel      map[uint64]bool
	drag     dragState
	draft    *scene.Object // shape being drawn
	polyPts  []geom.Vec    // in-progress polygon vertices
	textObj  *scene.Object // text being edited (not yet in doc)
	editText *scene.Object // existing text object being re-edited

	clipboard []*scene.Object

	showGrid bool
	snap     bool

	overlay   overlayKind
	palQuery  string
	palSel    int
	colorSel  int
	layerSel  int
	helpTop   int
	prompt    *promptState
	statusMsg string
	statusLvl statusLevel
	statusAt  time.Time

	mouseCell  struct{ X, Y int }
	mouseWorld geom.Vec

	toolbarHits []hitRegion // rebuilt every frame
}

type hitRegion struct {
	x0, x1 int
	action func(m *Model) tea.Cmd
}

// New builds the editor model, optionally loading a file.
func New(path string) (*Model, error) {
	m := &Model{
		doc:         scene.NewDoc(),
		theme:       DefaultTheme,
		tool:        ToolSelect,
		strokeIdx:   1,
		fillIdx:     3,
		strokeWidth: 1,
		opacity:     1,
		sel:         map[uint64]bool{},
		showGrid:    true,
	}
	if path != "" {
		if _, err := os.Stat(path); err == nil {
			doc, err := scene.Load(path)
			if err != nil {
				return nil, err
			}
			m.doc = doc
		}
		m.path = path
	}
	return m, nil
}

type autosaveTick time.Time

func autosaveCmd() tea.Cmd {
	return tea.Tick(30*time.Second, func(t time.Time) tea.Msg { return autosaveTick(t) })
}

func (m *Model) Init() tea.Cmd { return autosaveCmd() }

// canvasRows is the number of terminal rows dedicated to the canvas.
func (m *Model) canvasRows() int { return maxInt(1, m.h-2) }

// view returns the current pixel-space View for the active render mode.
func (m *Model) view() render.View {
	sx, sy := m.mode.PixelScale()
	factor := 1.0
	if m.mode == render.ModeBraille {
		factor = 2
	}
	return render.View{
		Center: m.doc.Camera.Center,
		Zoom:   m.doc.Camera.Zoom * factor,
		W:      m.w * sx,
		H:      m.canvasRows() * sy,
	}
}

// cellToWorld converts a terminal cell position to world coordinates.
func (m *Model) cellToWorld(cx, cy int) geom.Vec {
	sx, sy := m.mode.PixelScale()
	px := geom.V(float64(cx*sx)+float64(sx)/2, float64((cy-1)*sy)+float64(sy)/2)
	return m.view().ToWorld(px)
}

// hitTolerance is the world-space slop for picking, ~3 pixels.
func (m *Model) hitTolerance() float64 { return 3 / m.view().Zoom }

// maybeSnap snaps p to the adaptive grid when snapping is on.
func (m *Model) maybeSnap(p geom.Vec) geom.Vec {
	if !m.snap {
		return p
	}
	step := render.GridStep(m.doc.Camera.Zoom)
	return geom.V(math.Round(p.X/step)*step, math.Round(p.Y/step)*step)
}

// checkpoint pushes an undo snapshot before a labeled mutation.
func (m *Model) checkpoint(label string) {
	m.hist.Push(m.doc, label)
	m.dirty = true
}

func (m *Model) selection() []*scene.Object {
	var out []*scene.Object
	for _, o := range m.doc.Objects {
		if m.sel[o.ID] {
			out = append(out, o)
		}
	}
	return out
}

func (m *Model) clearSelection() { m.sel = map[uint64]bool{} }

func (m *Model) selectionBounds() (geom.Rect, bool) {
	objs := m.selection()
	if len(objs) == 0 {
		return geom.Rect{}, false
	}
	b := objs[0].Bounds()
	for _, o := range objs[1:] {
		b = b.Union(o.Bounds())
	}
	return b, true
}

func (m *Model) setStatus(lvl statusLevel, format string, args ...any) {
	m.statusMsg = fmt.Sprintf(format, args...)
	m.statusLvl = lvl
	m.statusAt = time.Now()
}

// newObject creates an object pre-configured with the current style.
func (m *Model) newObject(kind scene.Kind) *scene.Object {
	return &scene.Object{
		Kind:        kind,
		Stroke:      Palette[m.strokeIdx],
		Fill:        Palette[m.fillIdx],
		Filled:      m.filled && (kind == scene.KindRect || kind == scene.KindEllipse || kind == scene.KindPolygon),
		StrokeWidth: m.strokeWidth,
		Opacity:     m.opacity,
		Dashed:      m.dashed,
		Layer:       m.currentLayer(),
	}
}

func (m *Model) currentLayer() int {
	if m.layerSel >= 0 && m.layerSel < len(m.doc.Layers) {
		return m.layerSel
	}
	return 0
}

// Update is the bubbletea event dispatcher.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		return m, nil
	case autosaveTick:
		if m.dirty && m.path != "" {
			if err := m.doc.Save(m.path + ".autosave"); err == nil {
				m.setStatus(statusInfo, "autosaved %s.autosave", m.path)
			}
		}
		return m, autosaveCmd()
	case tea.MouseMsg:
		return m, m.handleMouse(msg)
	case tea.KeyMsg:
		// Stale status messages clear on the next keypress.
		if m.statusMsg != "" && time.Since(m.statusAt) > 4*time.Second {
			m.statusMsg = ""
		}
		if m.prompt != nil {
			return m, m.handlePromptKey(msg)
		}
		if m.textObj != nil {
			return m, m.handleTextKey(msg)
		}
		if m.overlay != ovNone {
			return m, m.handleOverlayKey(msg)
		}
		return m, m.handleKey(msg)
	}
	return m, nil
}

// --- small helpers ---

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func clampInt(v, lo, hi int) int { return maxInt(lo, minInt(hi, v)) }

func paletteIndexOf(c scene.Color) int {
	for i, p := range Palette {
		if p == c {
			return i
		}
	}
	return -1
}
