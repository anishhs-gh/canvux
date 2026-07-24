// Package app implements the interactive Canvux terminal editor.
package app

import (
	"fmt"
	"math"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/anishhs-gh/canvux/internal/collab"
	"github.com/anishhs-gh/canvux/internal/config"
	"github.com/anishhs-gh/canvux/internal/geom"
	"github.com/anishhs-gh/canvux/internal/history"
	"github.com/anishhs-gh/canvux/internal/plugin"
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
	ToolBezier
	ToolText
	ToolEraser
)

var toolNames = map[Tool]string{
	ToolSelect: "select", ToolPan: "pan", ToolBrush: "brush", ToolLine: "line",
	ToolRect: "rect", ToolEllipse: "ellipse", ToolArrow: "arrow",
	ToolPolygon: "polygon", ToolBezier: "curve", ToolText: "text", ToolEraser: "eraser",
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
	ovStencils
	ovOutline
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

	w, h    int
	mode    render.Mode
	theme   Theme
	profile render.Profile

	tool        Tool
	prevTool    Tool
	strokeIdx   int
	fillIdx     int
	filled      bool
	strokeWidth float64
	dashed      bool
	opacity     float64
	shadow      bool
	varBrush    bool // speed-sensitive variable-width brush

	sel       map[uint64]bool
	drag      dragState
	draft     *scene.Object // shape being drawn
	polyPts   []geom.Vec    // in-progress polygon vertices
	textObj   *scene.Object // text being edited (not yet in doc)
	editText  *scene.Object // existing text object being re-edited
	textCaret int           // caret position (rune index) within textObj.Text

	clipboard []*scene.Object

	showGrid bool
	snap     bool

	overlay    overlayKind
	palQuery   string
	palSel     int
	colorSel   int
	layerSel   int
	helpTop    int
	stencilSel int
	stencilQry string
	outlineSel int
	outlineQry string
	prompt     *promptState
	statusMsg  string
	statusLvl  statusLevel
	statusAt   time.Time

	// Rebindable keymap (key string -> action ID) and theme/palette cursors.
	keymap       map[string]string
	themeIdx     int
	paletteIdx   int
	autosaveSecs int

	mouseCell  struct{ X, Y int }
	mouseWorld geom.Vec

	toolbarHits []hitRegion // rebuilt every frame

	plugins []plugin.Manifest

	collab       *collab.Client
	peers        map[int]peerState
	collabShadow map[uint64]string

	// Presentation mode: layers become slides.
	present    bool
	presentIdx int

	antPhase   int  // marching-ants animation frame
	antRunning bool // ant heartbeat currently scheduled

	alignGuides []guideLine // smart-alignment guides for the current drag

	// Reusable render buffers, resized in place to avoid per-frame allocation.
	pb   *render.PixelBuf
	grid *render.CellGrid

	// pendingOSC is a one-shot terminal escape (e.g. OSC 52 clipboard) emitted
	// with the next frame, then cleared.
	pendingOSC string

	// Keyboard-only drawing: a virtual cursor placed with the arrow keys.
	kbDraw   bool
	kbCursor geom.Vec
}

type hitRegion struct {
	x0, x1 int
	action func(m *Model) tea.Cmd
}

// SetColorProfile sets the terminal color capability used when serializing
// frames. Defaults to auto-detection when unset.
func (m *Model) SetColorProfile(p render.Profile) { m.profile = p }

// New builds the editor model, optionally loading a file. User config is
// applied on top of the built-in defaults.
func New(path string) (*Model, error) {
	m := &Model{
		doc:          scene.NewDoc(),
		theme:        DefaultTheme,
		profile:      render.DetectProfile(),
		tool:         ToolSelect,
		strokeIdx:    1,
		fillIdx:      3,
		strokeWidth:  1,
		opacity:      1,
		sel:          map[uint64]bool{},
		showGrid:     true,
		keymap:       DefaultKeymap(),
		autosaveSecs: 30,
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
	cfg, err := config.Load()
	if err != nil {
		// A broken config shouldn't stop the editor; surface it in the status bar.
		m.setStatus(statusErr, "config: %v", err)
	} else {
		m.applyConfig(cfg)
	}
	m.plugins = plugin.Discover()
	m.checkAutosaveRecovery()
	return m, nil
}

// checkAutosaveRecovery prompts the user to recover from <file>.autosave when
// one exists and is newer than the file on disk (e.g. after a crash).
func (m *Model) checkAutosaveRecovery() {
	if m.path == "" {
		return
	}
	auto := m.path + ".autosave"
	ai, err := os.Stat(auto)
	if err != nil {
		return
	}
	// Newer than the main file (or the main file doesn't exist yet).
	if fi, err := os.Stat(m.path); err == nil && !ai.ModTime().After(fi.ModTime()) {
		return
	}
	m.prompt = &promptState{
		label: "Recover newer autosave for this file? [y/n]",
		yesNo: true,
		onYes: func(m *Model) tea.Cmd {
			doc, err := scene.Load(auto)
			if err != nil {
				m.setStatus(statusErr, "recover failed: %v", err)
				return nil
			}
			m.doc = doc
			m.dirty = true // recovered content isn't saved to the main file yet
			m.setStatus(statusOK, "recovered from %s — save to keep", auto)
			return nil
		},
		onNo: func(m *Model) tea.Cmd {
			os.Remove(auto)
			m.setStatus(statusInfo, "discarded autosave")
			return nil
		},
	}
}

// applyConfig folds loaded user preferences into the model.
func (m *Model) applyConfig(cfg config.Config) {
	if cfg.Theme != "" {
		if th, ok := ThemeByName(cfg.Theme); ok {
			m.theme = th
			for i, t := range Themes {
				if t.Name == cfg.Theme {
					m.themeIdx = i
				}
			}
		}
	}
	if cfg.Palette != "" && SetActivePalette(cfg.Palette) {
		for i, p := range Palettes {
			if p.Name == cfg.Palette {
				m.paletteIdx = i
			}
		}
	}
	if cfg.RenderMode == "braille" {
		m.mode = render.ModeBraille
	} else if cfg.RenderMode == "block" {
		m.mode = render.ModeHalfBlock
	}
	if cfg.Color != "" {
		if p, ok := render.ParseProfile(cfg.Color); ok {
			m.profile = p
		}
	}
	if cfg.Grid != nil {
		m.showGrid = *cfg.Grid
	}
	if cfg.Snap != nil {
		m.snap = *cfg.Snap
	}
	if cfg.Autosave != nil && *cfg.Autosave > 0 {
		m.autosaveSecs = *cfg.Autosave
	}
	for key, id := range cfg.Keys {
		if _, ok := actionByID(id); ok {
			m.keymap[key] = id
		}
	}
	m.strokeIdx = clampInt(m.strokeIdx, 0, len(Palette)-1)
	m.fillIdx = clampInt(m.fillIdx, 0, len(Palette)-1)
}

type autosaveTick time.Time

func autosaveCmd(secs int) tea.Cmd {
	if secs <= 0 {
		secs = 30
	}
	return tea.Tick(time.Duration(secs)*time.Second, func(t time.Time) tea.Msg { return autosaveTick(t) })
}

// antTick drives the marching-ants selection animation.
type antTick time.Time

func antTickCmd() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(t time.Time) tea.Msg { return antTick(t) })
}

// wantsAnimation reports whether the ant heartbeat needs to keep ticking.
func (m *Model) wantsAnimation() bool {
	return len(m.sel) > 0 || m.drag.kind == dragMarquee || m.draft != nil || len(m.polyPts) > 0
}

func (m *Model) Init() tea.Cmd {
	m.antRunning = true
	cmds := []tea.Cmd{autosaveCmd(m.autosaveSecs), antTickCmd(), tea.SetWindowTitle(m.windowTitle())}
	if m.collab != nil {
		cmds = append(cmds, collabTickCmd())
	}
	return tea.Batch(cmds...)
}

// windowTitle names the terminal window/tab after the open file.
func (m *Model) windowTitle() string {
	name := m.path
	if name == "" {
		name = "untitled"
	}
	return "Canvux — " + name
}

// setTitleCmd returns a command that refreshes the window title.
func (m *Model) setTitleCmd() tea.Cmd { return tea.SetWindowTitle(m.windowTitle()) }

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
		Shadow:      m.shadow,
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
		return m, autosaveCmd(m.autosaveSecs)
	case collabTick:
		if m.collab == nil {
			return m, nil
		}
		m.syncCollab()
		return m, collabTickCmd()
	case pluginResult:
		m.handlePluginResult(msg)
		return m, nil
	case clipboardResult:
		m.handleClipboardResult(msg)
		return m, nil
	case antTick:
		m.antPhase++
		if m.wantsAnimation() {
			return m, antTickCmd()
		}
		m.antRunning = false
		return m, nil
	case tea.MouseMsg:
		if m.present {
			return m, m.presentMouse(msg)
		}
		return m, m.withAnimation(m.handleMouse(msg))
	case tea.KeyMsg:
		// Stale status messages clear on the next keypress.
		if m.statusMsg != "" && time.Since(m.statusAt) > 4*time.Second {
			m.statusMsg = ""
		}
		if m.present {
			return m, m.presentKey(msg.String())
		}
		if m.prompt != nil {
			return m, m.withAnimation(m.handlePromptKey(msg))
		}
		if m.textObj != nil {
			return m, m.withAnimation(m.handleTextKey(msg))
		}
		if m.overlay != ovNone {
			return m, m.withAnimation(m.handleOverlayKey(msg))
		}
		return m, m.withAnimation(m.handleKey(msg))
	}
	return m, nil
}

// withAnimation restarts the marching-ants heartbeat if an interaction just
// created something animatable and the ticker had stopped.
func (m *Model) withAnimation(cmd tea.Cmd) tea.Cmd {
	if m.wantsAnimation() && !m.antRunning {
		m.antRunning = true
		return tea.Batch(cmd, antTickCmd())
	}
	return cmd
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
