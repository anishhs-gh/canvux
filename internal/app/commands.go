package app

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/anishhs-gh/canvux/internal/export"
	"github.com/anishhs-gh/canvux/internal/geom"
	"github.com/anishhs-gh/canvux/internal/history"
	"github.com/anishhs-gh/canvux/internal/imgembed"
	"github.com/anishhs-gh/canvux/internal/render"
	"github.com/anishhs-gh/canvux/internal/scene"
	"github.com/anishhs-gh/canvux/internal/svg"
)

// Command is one row in the command palette: a label, its current key hint,
// and the action to run.
type Command struct {
	Name string
	Keys string
	Do   func(m *Model) tea.Cmd
}

// commands builds the palette rows from the shared action table, filling each
// key hint from the *active* keymap so custom rebindings are reflected.
func (m *Model) commands() []Command {
	table := actionTable()
	out := make([]Command, 0, len(table))
	for _, a := range table {
		out = append(out, Command{Name: a.Name, Keys: m.keyForAction(a.ID), Do: a.Do})
	}
	return out
}

// cmdCycleTheme steps dark → light → high-contrast.
func (m *Model) cmdCycleTheme() tea.Cmd {
	m.themeIdx = (m.themeIdx + 1) % len(Themes)
	m.theme = Themes[m.themeIdx].Theme
	m.setStatus(statusInfo, "theme: %s", Themes[m.themeIdx].Name)
	return nil
}

// cmdCyclePalette steps the drawing palette default → colorblind → high-contrast.
func (m *Model) cmdCyclePalette() tea.Cmd {
	m.paletteIdx = (m.paletteIdx + 1) % len(Palettes)
	SetActivePalette(Palettes[m.paletteIdx].Name)
	m.strokeIdx = clampInt(m.strokeIdx, 0, len(Palette)-1)
	m.fillIdx = clampInt(m.fillIdx, 0, len(Palette)-1)
	m.setStatus(statusInfo, "palette: %s", Palettes[m.paletteIdx].Name)
	return nil
}

func (m *Model) setTool(t Tool) {
	if m.tool == ToolPolygon && t != ToolPolygon {
		m.finishPolygon()
	}
	m.tool = t
	m.setStatus(statusInfo, "tool: %s", toolNames[t])
}

// --- file commands ---

func (m *Model) cmdSave() tea.Cmd {
	if m.path == "" {
		return m.cmdSaveAs()
	}
	return m.saveTo(m.path)
}

func (m *Model) cmdSaveAs() tea.Cmd {
	def := m.path
	if def == "" {
		def = "drawing.canvux"
	}
	m.prompt = &promptState{
		label: "Save as", value: def,
		confirm: func(m *Model, v string) tea.Cmd { return m.saveTo(v) },
	}
	return nil
}

func (m *Model) saveTo(path string) tea.Cmd {
	if path == "" {
		return nil
	}
	if filepath.Ext(path) == "" {
		path += ".canvux"
	}
	if err := m.doc.Save(path); err != nil {
		m.setStatus(statusErr, "save failed: %v", err)
		return nil
	}
	m.path = path
	m.dirty = false
	os.Remove(path + ".autosave")
	m.setStatus(statusOK, "saved %s (%d objects)", path, len(m.doc.Objects))
	return m.setTitleCmd()
}

func (m *Model) cmdOpen() tea.Cmd {
	open := func(m *Model, v string) tea.Cmd {
		if v == "" {
			return nil
		}
		doc, err := scene.Load(v)
		if err != nil {
			m.setStatus(statusErr, "open failed: %v", err)
			return nil
		}
		m.doc = doc
		m.path = v
		m.dirty = false
		m.hist = history.Stack{}
		m.clearSelection()
		if fi, err := os.Stat(v + ".autosave"); err == nil {
			m.setStatus(statusInfo, "opened %s — autosave from %s exists (%s.autosave)",
				v, fi.ModTime().Format("15:04"), v)
		} else {
			m.setStatus(statusOK, "opened %s (%d objects)", v, len(doc.Objects))
		}
		return tea.Batch(m.cmdZoomFit(), m.setTitleCmd())
	}
	prompt := func(m *Model) tea.Cmd {
		m.prompt = &promptState{label: "Open file", value: "", confirm: open}
		return nil
	}
	return m.confirmIfDirty("Discard unsaved changes and open another file?", prompt)
}

func (m *Model) cmdNew() tea.Cmd {
	reset := func(m *Model) tea.Cmd {
		m.doc = scene.NewDoc()
		m.path = ""
		m.dirty = false
		m.hist = history.Stack{}
		m.clearSelection()
		m.setStatus(statusOK, "new document")
		return m.setTitleCmd()
	}
	return m.confirmIfDirty("Discard unsaved changes and start new?", reset)
}

func (m *Model) confirmIfDirty(q string, action func(m *Model) tea.Cmd) tea.Cmd {
	if !m.dirty {
		return action(m)
	}
	m.prompt = &promptState{label: q + " [y/n]", yesNo: true, onYes: action}
	return nil
}

func (m *Model) cmdExport() tea.Cmd {
	def := "drawing.svg"
	if m.path != "" {
		def = strings.TrimSuffix(m.path, filepath.Ext(m.path)) + ".svg"
	}
	m.prompt = &promptState{
		label: "Export to (.svg or .png)", value: def,
		confirm: func(m *Model, v string) tea.Cmd {
			if v == "" {
				return nil
			}
			var err error
			switch strings.ToLower(filepath.Ext(v)) {
			case ".png":
				err = ExportPNG(m.doc, v, 8)
			default:
				err = os.WriteFile(v, svg.Export(m.doc), 0o644)
			}
			if err != nil {
				m.setStatus(statusErr, "export failed: %v", err)
			} else {
				m.setStatus(statusOK, "exported %s", v)
			}
			return nil
		},
	}
	return nil
}

// ExportPNG renders doc to a PNG file at scale pixels per world unit.
func ExportPNG(d *scene.Doc, path string, scale float64) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return export.PNG(f, d, scale, DefaultTheme.CanvasBG)
}

func (m *Model) cmdImportSVG() tea.Cmd {
	m.prompt = &promptState{
		label: "Import SVG file", value: "",
		confirm: func(m *Model, v string) tea.Cmd {
			if v == "" {
				return nil
			}
			data, err := os.ReadFile(v)
			if err != nil {
				m.setStatus(statusErr, "import failed: %v", err)
				return nil
			}
			imported, skipped, err := svg.Import(data)
			if err != nil {
				m.setStatus(statusErr, "import failed: %v", err)
				return nil
			}
			m.checkpoint("import svg")
			m.clearSelection()
			for _, o := range imported.Objects {
				c := o.Clone()
				c.Layer = m.currentLayer()
				m.doc.Add(c)
				m.sel[c.ID] = true
			}
			note := ""
			if skipped > 0 {
				note = fmt.Sprintf(" (%d unsupported elements skipped)", skipped)
			}
			m.setStatus(statusOK, "imported %d objects%s", len(imported.Objects), note)
			return m.cmdZoomFit()
		},
	}
	return nil
}

// --- edit commands ---

func (m *Model) cmdUndo() tea.Cmd {
	if doc, label, ok := m.hist.Undo(m.doc); ok {
		m.doc = doc
		m.dirty = true
		m.pruneSelection()
		m.setStatus(statusInfo, "undo: %s", label)
	} else {
		m.setStatus(statusInfo, "nothing to undo")
	}
	return nil
}

func (m *Model) cmdRedo() tea.Cmd {
	if doc, label, ok := m.hist.Redo(m.doc); ok {
		m.doc = doc
		m.dirty = true
		m.pruneSelection()
		m.setStatus(statusInfo, "redo: %s", label)
	} else {
		m.setStatus(statusInfo, "nothing to redo")
	}
	return nil
}

func (m *Model) pruneSelection() {
	for id := range m.sel {
		if m.doc.Get(id) == nil {
			delete(m.sel, id)
		}
	}
}

func (m *Model) cmdCopy() tea.Cmd {
	objs := m.selection()
	if len(objs) == 0 {
		m.setStatus(statusInfo, "nothing selected")
		return nil
	}
	m.clipboard = nil
	for _, o := range objs {
		m.clipboard = append(m.clipboard, o.Clone())
	}
	m.setStatus(statusOK, "copied %d object(s)", len(objs))
	return nil
}

func (m *Model) cmdPaste() tea.Cmd {
	if len(m.clipboard) == 0 {
		m.setStatus(statusInfo, "clipboard empty")
		return nil
	}
	m.checkpoint("paste")
	m.clearSelection()
	off := geom.V(2, 2)
	for _, o := range m.clipboard {
		c := o.Clone()
		c.Translate(off)
		m.doc.Add(c)
		m.sel[c.ID] = true
	}
	m.setStatus(statusOK, "pasted %d object(s)", len(m.clipboard))
	return nil
}

func (m *Model) cmdDuplicate() tea.Cmd {
	objs := m.selection()
	if len(objs) == 0 {
		m.setStatus(statusInfo, "nothing selected")
		return nil
	}
	m.checkpoint("duplicate")
	m.clearSelection()
	for _, o := range objs {
		c := o.Clone()
		c.Translate(geom.V(2, 2))
		m.doc.Add(c)
		m.sel[c.ID] = true
	}
	m.setStatus(statusOK, "duplicated %d object(s)", len(objs))
	return nil
}

func (m *Model) cmdDelete() tea.Cmd {
	objs := m.selection()
	if len(objs) == 0 {
		return nil
	}
	m.checkpoint("delete")
	for _, o := range objs {
		m.doc.Remove(o.ID)
	}
	m.clearSelection()
	m.setStatus(statusOK, "deleted %d object(s)", len(objs))
	return nil
}

func (m *Model) cmdSelectAll() tea.Cmd {
	m.clearSelection()
	for _, o := range m.doc.VisibleObjects() {
		m.sel[o.ID] = true
	}
	m.setStatus(statusInfo, "%d selected", len(m.sel))
	return nil
}

func (m *Model) cmdZ(raise bool) tea.Cmd {
	objs := m.selection()
	if len(objs) == 0 {
		return nil
	}
	m.checkpoint("z-order")
	for _, o := range objs {
		if raise {
			m.doc.Raise(o.ID)
		} else {
			m.doc.Lower(o.ID)
		}
	}
	return nil
}

func (m *Model) cmdRotate(deg float64) tea.Cmd {
	objs := m.selection()
	if len(objs) == 0 {
		return nil
	}
	m.checkpoint("rotate")
	for _, o := range objs {
		o.Rotation = math.Mod(o.Rotation+deg*math.Pi/180, 2*math.Pi)
	}
	m.setStatus(statusInfo, "rotated %+.0f°", deg)
	return nil
}

// applyStyle mutates selection style (with one checkpoint) or, with no
// selection, updates the current tool style.
func (m *Model) applyStyle(label string, fn func(o *scene.Object)) {
	objs := m.selection()
	if len(objs) == 0 {
		return
	}
	m.checkpoint(label)
	for _, o := range objs {
		fn(o)
	}
}

func (m *Model) cmdToggleFill() tea.Cmd {
	m.filled = !m.filled
	m.applyStyle("fill", func(o *scene.Object) {
		if o.Kind == scene.KindRect || o.Kind == scene.KindEllipse || o.Kind == scene.KindPolygon {
			o.Filled = m.filled
			o.Fill = Palette[m.fillIdx]
		}
	})
	m.setStatus(statusInfo, "fill: %v", m.filled)
	return nil
}

func (m *Model) cmdToggleDash() tea.Cmd {
	m.dashed = !m.dashed
	m.applyStyle("dash", func(o *scene.Object) { o.Dashed = m.dashed })
	m.setStatus(statusInfo, "dashed: %v", m.dashed)
	return nil
}

func (m *Model) cmdToggleShadow() tea.Cmd {
	m.shadow = !m.shadow
	m.applyStyle("shadow", func(o *scene.Object) { o.Shadow = m.shadow })
	m.setStatus(statusInfo, "shadow: %v", m.shadow)
	return nil
}

func (m *Model) cmdBlur(d float64) tea.Cmd {
	if len(m.sel) == 0 {
		m.setStatus(statusInfo, "select objects to blur")
		return nil
	}
	m.applyStyle("blur", func(o *scene.Object) {
		o.Blur = math.Max(0, math.Min(4, o.Blur+d))
	})
	if objs := m.selection(); len(objs) > 0 {
		m.setStatus(statusInfo, "blur: %.1f", objs[0].Blur)
	}
	return nil
}

func (m *Model) cmdGradAngle() tea.Cmd {
	if len(m.sel) == 0 {
		m.setStatus(statusInfo, "select gradient objects first")
		return nil
	}
	m.applyStyle("gradient angle", func(o *scene.Object) {
		if o.Fill2 != nil {
			o.GradAngle = math.Mod(o.GradAngle+45, 360)
		}
	})
	m.setStatus(statusInfo, "gradient rotated +45°")
	return nil
}

func (m *Model) cmdStrokeWidth(d float64) tea.Cmd {
	m.strokeWidth = math.Max(0.5, math.Min(8, m.strokeWidth+d))
	m.applyStyle("stroke width", func(o *scene.Object) { o.StrokeWidth = m.strokeWidth })
	m.setStatus(statusInfo, "stroke width: %.1f", m.strokeWidth)
	return nil
}

func (m *Model) cmdOpacity(d float64) tea.Cmd {
	m.opacity = math.Max(0.1, math.Min(1, m.opacity+d))
	m.applyStyle("opacity", func(o *scene.Object) { o.Opacity = m.opacity })
	m.setStatus(statusInfo, "opacity: %.0f%%", m.opacity*100)
	return nil
}

func (m *Model) cmdZoomFit() tea.Cmd {
	b := m.doc.ContentBounds()
	if b.W() <= 0 && b.H() <= 0 {
		m.doc.Camera = scene.Camera{Zoom: 2}
		return nil
	}
	b = b.Expand(2)
	v := m.view()
	zx := float64(v.W) / math.Max(b.W(), 0.001)
	zy := float64(v.H) / math.Max(b.H(), 0.001)
	factor := 1.0
	if m.mode == render.ModeBraille {
		factor = 2
	}
	m.doc.Camera.Zoom = math.Max(0.05, math.Min(400, math.Min(zx, zy)/factor))
	m.doc.Camera.Center = b.Center()
	return nil
}

func (m *Model) cmdToggleMode() tea.Cmd {
	if m.mode == render.ModeHalfBlock {
		m.mode = render.ModeBraille
	} else {
		m.mode = render.ModeHalfBlock
	}
	m.setStatus(statusInfo, "render mode: %s", m.mode)
	return nil
}

func (m *Model) cmdQuit() tea.Cmd {
	quit := func(m *Model) tea.Cmd {
		if m.collab != nil {
			m.collab.Close()
		}
		return tea.Quit
	}
	if m.collab != nil {
		// Collab docs live on the server; no unsaved-changes prompt.
		return quit(m)
	}
	return m.confirmIfDirty("Quit without saving?", quit)
}

func (m *Model) cmdImportImage() tea.Cmd {
	m.prompt = &promptState{
		label: "Image file (png/jpg/gif)", value: "",
		confirm: func(m *Model, v string) tea.Cmd {
			if v == "" {
				return nil
			}
			objs, err := imgembed.FromFile(v, 48, m.currentLayer())
			if err != nil {
				m.setStatus(statusErr, "image import failed: %v", err)
				return nil
			}
			m.checkpoint("import image")
			m.clearSelection()
			off := m.doc.Camera.Center
			for _, o := range objs {
				o.Translate(off)
				m.doc.Add(o)
				m.sel[o.ID] = true
			}
			m.setStatus(statusOK, "imported image as %d rects — drag to place", len(objs))
			return nil
		},
	}
	return nil
}
