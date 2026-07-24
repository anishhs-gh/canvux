package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/anishhs-gh/canvux/internal/scene"
)

// Action is a named, palette-invokable and keybindable command. It is the
// single source of truth shared by the command palette and the keymap, so a
// binding and its palette entry can never drift apart.
type Action struct {
	ID   string // stable identity, e.g. "tool.rect"; used by keymaps
	Name string // human label shown in the palette
	Do   func(m *Model) tea.Cmd
}

// actionTable is the master registry, in palette display order.
func actionTable() []Action {
	tool := func(t Tool) func(*Model) tea.Cmd {
		return func(m *Model) tea.Cmd { m.setTool(t); return nil }
	}
	return []Action{
		{"tool.select", "Tool: Select", tool(ToolSelect)},
		{"tool.pan", "Tool: Pan", tool(ToolPan)},
		{"tool.brush", "Tool: Brush", tool(ToolBrush)},
		{"tool.line", "Tool: Line", tool(ToolLine)},
		{"tool.rect", "Tool: Rectangle", tool(ToolRect)},
		{"tool.ellipse", "Tool: Ellipse", tool(ToolEllipse)},
		{"tool.arrow", "Tool: Arrow", tool(ToolArrow)},
		{"tool.polygon", "Tool: Polygon", tool(ToolPolygon)},
		{"tool.bezier", "Tool: Curve (bézier)", tool(ToolBezier)},
		{"tool.text", "Tool: Text", tool(ToolText)},
		{"tool.eraser", "Tool: Eraser", tool(ToolEraser)},

		{"file.save", "Save", (*Model).cmdSave},
		{"file.saveas", "Save As…", (*Model).cmdSaveAs},
		{"file.open", "Open…", (*Model).cmdOpen},
		{"file.export", "Export SVG / PNG…", (*Model).cmdExport},
		{"file.import-svg", "Import SVG…", (*Model).cmdImportSVG},
		{"file.import-image", "Import Image (pixel grid)…", (*Model).cmdImportImage},
		{"file.new", "New Document", (*Model).cmdNew},

		{"edit.undo", "Undo", (*Model).cmdUndo},
		{"edit.redo", "Redo", (*Model).cmdRedo},
		{"edit.copy", "Copy Selection", (*Model).cmdCopy},
		{"edit.copy-svg", "Copy as SVG to System Clipboard", (*Model).cmdCopySVG},
		{"edit.paste", "Paste", (*Model).cmdPaste},
		{"edit.duplicate", "Duplicate Selection", (*Model).cmdDuplicate},
		{"edit.delete", "Delete Selection", (*Model).cmdDelete},
		{"edit.select-all", "Select All", (*Model).cmdSelectAll},
		{"edit.raise", "Raise (z-order)", func(m *Model) tea.Cmd { return m.cmdZ(true) }},
		{"edit.lower", "Lower (z-order)", func(m *Model) tea.Cmd { return m.cmdZ(false) }},
		{"edit.rotate-cw", "Rotate +15°", func(m *Model) tea.Cmd { return m.cmdRotate(15) }},
		{"edit.rotate-ccw", "Rotate -15°", func(m *Model) tea.Cmd { return m.cmdRotate(-15) }},
		{"edit.rotate-cw-fine", "Rotate +1°", func(m *Model) tea.Cmd { return m.cmdRotate(1) }},
		{"edit.rotate-ccw-fine", "Rotate -1°", func(m *Model) tea.Cmd { return m.cmdRotate(-1) }},

		{"style.stroke-color", "Stroke Color…", func(m *Model) tea.Cmd {
			m.overlay, m.colorSel = ovColorStroke, m.strokeIdx
			return nil
		}},
		{"style.fill-color", "Fill Color…", func(m *Model) tea.Cmd {
			m.overlay, m.colorSel = ovColorFill, m.fillIdx
			return nil
		}},
		{"style.toggle-fill", "Toggle Fill", (*Model).cmdToggleFill},
		{"style.toggle-dash", "Toggle Dashed", (*Model).cmdToggleDash},
		{"style.toggle-shadow", "Toggle Shadow", (*Model).cmdToggleShadow},
		{"style.blur-more", "Blur +", func(m *Model) tea.Cmd { return m.cmdBlur(0.5) }},
		{"style.blur-less", "Blur -", func(m *Model) tea.Cmd { return m.cmdBlur(-0.5) }},
		{"style.gradient-pick", "Gradient: Pick End Color…", func(m *Model) tea.Cmd {
			m.overlay, m.colorSel = ovColorFill, m.fillIdx
			m.setStatus(statusInfo, "pick a color, press g to set it as the gradient end")
			return nil
		}},
		{"style.gradient-clear", "Gradient: Clear", func(m *Model) tea.Cmd {
			m.applyStyle("clear gradient", func(o *scene.Object) { o.Fill2 = nil })
			return nil
		}},
		{"style.gradient-rotate", "Gradient: Rotate +45°", (*Model).cmdGradAngle},
		{"style.stroke-width-more", "Stroke Width +", func(m *Model) tea.Cmd { return m.cmdStrokeWidth(0.5) }},
		{"style.stroke-width-less", "Stroke Width -", func(m *Model) tea.Cmd { return m.cmdStrokeWidth(-0.5) }},
		{"style.opacity-more", "Opacity +10%", func(m *Model) tea.Cmd { return m.cmdOpacity(0.1) }},
		{"style.opacity-less", "Opacity -10%", func(m *Model) tea.Cmd { return m.cmdOpacity(-0.1) }},
		{"brush.toggle-variable", "Brush: Toggle Variable Width", func(m *Model) tea.Cmd {
			m.varBrush = !m.varBrush
			m.setStatus(statusInfo, "variable-width brush: %v", m.varBrush)
			return nil
		}},

		{"view.zoom-in", "Zoom In", func(m *Model) tea.Cmd { m.zoomAt(m.doc.Camera.Center, 1.25); return nil }},
		{"view.zoom-out", "Zoom Out", func(m *Model) tea.Cmd { m.zoomAt(m.doc.Camera.Center, 0.8); return nil }},
		{"view.zoom-100", "Zoom 100%", func(m *Model) tea.Cmd {
			m.doc.Camera.Zoom = 2
			m.setStatus(statusInfo, "zoom 100%%")
			return nil
		}},
		{"view.zoom-fit", "Zoom to Fit", (*Model).cmdZoomFit},
		{"view.toggle-grid", "Toggle Grid", func(m *Model) tea.Cmd { m.showGrid = !m.showGrid; return nil }},
		{"view.toggle-snap", "Toggle Snap to Grid", func(m *Model) tea.Cmd {
			m.snap = !m.snap
			m.setStatus(statusInfo, "snap: %v", m.snap)
			return nil
		}},
		{"view.toggle-mode", "Toggle Render Mode (block/braille)", (*Model).cmdToggleMode},
		{"view.cycle-palette", "Cycle Drawing Palette", (*Model).cmdCyclePalette},
		{"view.cycle-theme", "Cycle Theme (dark/light/high-contrast)", (*Model).cmdCycleTheme},

		{"panel.stencils", "Insert Stencil…", func(m *Model) tea.Cmd {
			m.overlay, m.stencilQry, m.stencilSel = ovStencils, "", 0
			return nil
		}},
		{"panel.layers", "Layers…", func(m *Model) tea.Cmd { m.overlay = ovLayers; return nil }},
		{"panel.outline", "Outline (object navigator)…", func(m *Model) tea.Cmd {
			m.overlay, m.outlineQry, m.outlineSel = ovOutline, "", 0
			return nil
		}},
		{"mode.present", "Presentation Mode (layers = slides)", (*Model).enterPresent},
		{"mode.kbdraw", "Keyboard Draw Mode (mouse-free, toggle)", (*Model).toggleKbDraw},
		{"help", "Help", func(m *Model) tea.Cmd { m.overlay, m.helpTop = ovHelp, 0; return nil }},
		{"quit", "Quit", (*Model).cmdQuit},
	}
}

// actionsByID indexes the table for keymap dispatch. Built once.
var actionIndex = func() map[string]Action {
	m := map[string]Action{}
	for _, a := range actionTable() {
		m[a.ID] = a
	}
	return m
}()

func actionByID(id string) (Action, bool) {
	a, ok := actionIndex[id]
	return a, ok
}

// DefaultKeymap maps a bubbletea key string to an action ID. It reproduces the
// historical single-key bindings; the config layer may override any entry.
// Context-sensitive keys (esc, enter, space, arrows, tab, digits) are handled
// directly in handleKey and intentionally excluded here.
func DefaultKeymap() map[string]string {
	return map[string]string{
		"v": "tool.select", "s": "tool.select",
		"b": "tool.brush", "l": "tool.line", "r": "tool.rect", "e": "tool.ellipse",
		"a": "tool.arrow", "p": "tool.polygon", "n": "tool.bezier", "t": "tool.text",
		"x": "tool.eraser",

		"+": "view.zoom-in", "=": "view.zoom-in", "-": "view.zoom-out", "_": "view.zoom-out",
		"0": "view.zoom-100", "F": "view.zoom-fit",
		"g": "view.toggle-grid", "G": "view.toggle-snap", "M": "view.toggle-mode",

		"u": "edit.undo", "ctrl+z": "edit.undo",
		"U": "edit.redo", "ctrl+y": "edit.redo", "ctrl+r": "edit.redo",
		"d": "edit.duplicate", "Y": "edit.copy", "V": "edit.paste", "ctrl+shift+c": "edit.copy-svg",
		"delete": "edit.delete", "backspace": "edit.delete", "ctrl+a": "edit.select-all",
		"]": "edit.raise", "[": "edit.lower",
		".": "edit.rotate-cw", ",": "edit.rotate-ccw", ">": "edit.rotate-cw-fine", "<": "edit.rotate-ccw-fine",

		"c": "style.stroke-color", "C": "style.fill-color",
		"f": "style.toggle-fill", "D": "style.toggle-dash", "S": "style.toggle-shadow",
		"B": "style.blur-more", "ctrl+b": "style.blur-less",
		"w": "style.stroke-width-more", "W": "style.stroke-width-less",
		"o": "style.opacity-less", "O": "style.opacity-more",

		"ctrl+s": "file.save", "ctrl+o": "file.open", "ctrl+e": "file.export",
		"L": "panel.layers", "i": "panel.stencils", "P": "mode.present",
		"K": "mode.kbdraw",
		"?": "help", "q": "quit",
	}
}

// keyForAction returns the shortest bound key for an action ID (for palette
// hints), or "" if unbound. Prefers a plain key over a ctrl/shift chord.
func (m *Model) keyForAction(id string) string {
	best := ""
	for key, aid := range m.keymap {
		if aid != id {
			continue
		}
		if best == "" || betterHint(key, best) {
			best = key
		}
	}
	return best
}

// betterHint prefers shorter, non-chord keys for display.
func betterHint(a, b string) bool {
	score := func(s string) int {
		n := len(s)
		if len(s) > 4 { // "ctrl+", "shift+" prefixes
			n += 10
		}
		return n
	}
	return score(a) < score(b)
}
