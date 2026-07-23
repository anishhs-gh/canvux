package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/anishhs-gh/canvux/internal/geom"
	"github.com/anishhs-gh/canvux/internal/scene"
)

func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()
	switch key {
	case "ctrl+c":
		return tea.Quit
	case "q":
		return m.cmdQuit()
	case "esc":
		switch {
		case m.polyPts != nil:
			m.polyPts = nil
			m.setStatus(statusInfo, "polygon cancelled")
		case m.draft != nil:
			m.draft = nil
			m.drag = dragState{}
		case len(m.sel) > 0:
			m.clearSelection()
		default:
			m.setTool(ToolSelect)
		}
		return nil
	case "enter":
		if m.tool == ToolPolygon && m.polyPts != nil {
			m.finishPolygon()
		}
		return nil

	// tools
	case "v", "s":
		m.setTool(ToolSelect)
	case " ", "space":
		if m.tool == ToolPan {
			m.setTool(m.prevTool)
		} else {
			m.prevTool = m.tool
			m.setTool(ToolPan)
		}
	case "b":
		m.setTool(ToolBrush)
	case "l":
		m.setTool(ToolLine)
	case "r":
		m.setTool(ToolRect)
	case "e":
		m.setTool(ToolEllipse)
	case "a":
		m.setTool(ToolArrow)
	case "p":
		m.setTool(ToolPolygon)
	case "n":
		m.setTool(ToolBezier)
	case "t":
		m.setTool(ToolText)
	case "x":
		m.setTool(ToolEraser)

	// view
	case "+", "=":
		m.zoomAt(m.doc.Camera.Center, 1.25)
	case "-", "_":
		m.zoomAt(m.doc.Camera.Center, 0.8)
	case "0":
		m.doc.Camera.Zoom = 2
		m.setStatus(statusInfo, "zoom 100%%")
	case "F":
		return m.cmdZoomFit()
	case "g":
		m.showGrid = !m.showGrid
	case "G":
		m.snap = !m.snap
		m.setStatus(statusInfo, "snap: %v", m.snap)
	case "M":
		return m.cmdToggleMode()

	// navigation / nudging
	case "up", "down", "left", "right", "h", "j", "k", "shift+up", "shift+down", "shift+left", "shift+right":
		return m.arrowKey(key)

	// edit
	case "u", "ctrl+z":
		return m.cmdUndo()
	case "U", "ctrl+y", "ctrl+r":
		return m.cmdRedo()
	case "d":
		return m.cmdDuplicate()
	case "Y":
		return m.cmdCopy()
	case "V":
		return m.cmdPaste()
	case "delete", "backspace":
		return m.cmdDelete()
	case "ctrl+a":
		return m.cmdSelectAll()
	case "tab":
		m.cycleSelection(1)
	case "shift+tab":
		m.cycleSelection(-1)
	case "]":
		return m.cmdZ(true)
	case "[":
		return m.cmdZ(false)
	case ".":
		return m.cmdRotate(15)
	case ",":
		return m.cmdRotate(-15)
	case ">":
		return m.cmdRotate(1)
	case "<":
		return m.cmdRotate(-1)

	// style
	case "c":
		m.overlay = ovColorStroke
		m.colorSel = m.strokeIdx
	case "C":
		m.overlay = ovColorFill
		m.colorSel = m.fillIdx
	case "f":
		return m.cmdToggleFill()
	case "D":
		return m.cmdToggleDash()
	case "S":
		return m.cmdToggleShadow()
	case "B":
		return m.cmdBlur(0.5)
	case "ctrl+b":
		return m.cmdBlur(-0.5)
	case "w":
		return m.cmdStrokeWidth(0.5)
	case "W":
		return m.cmdStrokeWidth(-0.5)
	case "o":
		return m.cmdOpacity(-0.1)
	case "O":
		return m.cmdOpacity(0.1)

	// files / UI
	case "ctrl+s":
		return m.cmdSave()
	case "ctrl+o":
		return m.cmdOpen()
	case "ctrl+e":
		return m.cmdExport()
	case "ctrl+p", ":":
		m.overlay = ovPalette
		m.palQuery = ""
		m.palSel = 0
	case "L":
		m.overlay = ovLayers
	case "i":
		m.overlay = ovStencils
		m.stencilQry = ""
		m.stencilSel = 0
	case "?":
		m.overlay = ovHelp
		m.helpTop = 0
	default:
		// digits select stroke colors directly
		if len(key) == 1 && key[0] >= '1' && key[0] <= '9' {
			idx := int(key[0] - '1')
			if idx < len(Palette) {
				m.strokeIdx = idx
				m.applyStyle("stroke color", func(o *scene.Object) { o.Stroke = Palette[idx] })
				m.setStatus(statusInfo, "stroke color %d", idx+1)
			}
		}
	}
	return nil
}

// arrowKey pans the camera, or nudges the selection when one exists.
func (m *Model) arrowKey(key string) tea.Cmd {
	dir := map[string]geom.Vec{
		"up": {X: 0, Y: -1}, "k": {X: 0, Y: -1}, "shift+up": {X: 0, Y: -1},
		"down": {X: 0, Y: 1}, "j": {X: 0, Y: 1}, "shift+down": {X: 0, Y: 1},
		"left": {X: -1, Y: 0}, "h": {X: -1, Y: 0}, "shift+left": {X: -1, Y: 0},
		"right": {X: 1, Y: 0}, "shift+right": {X: 1, Y: 0},
	}[key]
	fine := strings.HasPrefix(key, "shift+")
	if len(m.sel) > 0 && m.tool == ToolSelect {
		step := 1.0
		if fine {
			step = 0.25
		}
		m.checkpoint("nudge")
		for _, o := range m.selection() {
			o.Translate(dir.Mul(step))
		}
		return nil
	}
	// Pan by ~4 cells.
	d := 8 / m.doc.Camera.Zoom
	m.doc.Camera.Center = m.doc.Camera.Center.Add(dir.Mul(d))
	return nil
}

// cycleSelection walks through visible objects one at a time.
func (m *Model) cycleSelection(delta int) {
	vis := m.doc.VisibleObjects()
	if len(vis) == 0 {
		return
	}
	cur := -1
	if len(m.sel) == 1 {
		for id := range m.sel {
			for i, o := range vis {
				if o.ID == id {
					cur = i
				}
			}
		}
	}
	next := (cur + delta + len(vis)) % len(vis)
	m.clearSelection()
	m.sel[vis[next].ID] = true
	m.setStatus(statusInfo, "selected %s #%d", vis[next].Kind, vis[next].ID)
}

// --- text editing mode ---

func (m *Model) handleTextKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		if m.editText != nil {
			// Restore the exact pre-edit document and drop the redo entry.
			if doc, _, ok := m.hist.Undo(m.doc); ok {
				m.doc = doc
				m.hist.DropRedo()
			}
		}
		m.textObj, m.editText = nil, nil
	case "enter":
		t := m.textObj
		wasEdit := m.editText != nil
		m.textObj, m.editText = nil, nil
		if t == nil || strings.TrimSpace(t.Text) == "" {
			return nil
		}
		if !wasEdit {
			m.checkpoint("add text")
		}
		m.doc.Add(t)
		m.clearSelection()
		m.sel[t.ID] = true
		m.setStatus(statusOK, "text: %q", t.Text)
	case "backspace":
		r := []rune(m.textObj.Text)
		if len(r) > 0 {
			m.textObj.Text = string(r[:len(r)-1])
		}
	default:
		if msg.Type == tea.KeySpace {
			m.textObj.Text += " "
		} else if msg.Type == tea.KeyRunes {
			m.textObj.Text += string(msg.Runes)
		}
	}
	return nil
}

// --- prompt mode ---

func (m *Model) handlePromptKey(msg tea.KeyMsg) tea.Cmd {
	p := m.prompt
	if p.yesNo {
		switch msg.String() {
		case "y", "Y", "enter":
			m.prompt = nil
			if p.onYes != nil {
				return p.onYes(m)
			}
		case "n", "N", "esc":
			m.prompt = nil
			if p.onNo != nil {
				return p.onNo(m)
			}
		}
		return nil
	}
	switch msg.String() {
	case "esc":
		m.prompt = nil
	case "enter":
		m.prompt = nil
		return p.confirm(m, strings.TrimSpace(p.value))
	case "backspace":
		r := []rune(p.value)
		if len(r) > 0 {
			p.value = string(r[:len(r)-1])
		}
	case "ctrl+u":
		p.value = ""
	default:
		if msg.Type == tea.KeyRunes {
			p.value += string(msg.Runes)
		} else if msg.Type == tea.KeySpace {
			p.value += " "
		}
	}
	return nil
}

// --- overlay mode (palette, help, colors, layers) ---

func (m *Model) handleOverlayKey(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()
	if key == "esc" {
		m.overlay = ovNone
		return nil
	}
	switch m.overlay {
	case ovPalette:
		return m.paletteKey(msg)
	case ovHelp:
		switch key {
		case "up", "k":
			m.helpTop = maxInt(0, m.helpTop-1)
		case "down", "j":
			m.helpTop++
		case "q", "?", "enter":
			m.overlay = ovNone
		}
	case ovColorStroke, ovColorFill:
		return m.colorKey(key)
	case ovLayers:
		return m.layersKey(key)
	case ovStencils:
		return m.stencilKey(msg)
	}
	return nil
}

func (m *Model) stencilKey(msg tea.KeyMsg) tea.Cmd {
	items := m.filteredStencils()
	switch msg.String() {
	case "up", "ctrl+k":
		m.stencilSel = maxInt(0, m.stencilSel-1)
	case "down", "ctrl+j":
		m.stencilSel = minInt(maxInt(0, len(items)-1), m.stencilSel+1)
	case "enter":
		m.overlay = ovNone
		if m.stencilSel < len(items) {
			return m.insertStencil(items[m.stencilSel])
		}
	case "backspace":
		r := []rune(m.stencilQry)
		if len(r) > 0 {
			m.stencilQry = string(r[:len(r)-1])
		}
		m.stencilSel = 0
	default:
		if msg.Type == tea.KeyRunes {
			m.stencilQry += string(msg.Runes)
			m.stencilSel = 0
		} else if msg.Type == tea.KeySpace {
			m.stencilQry += " "
			m.stencilSel = 0
		}
	}
	return nil
}

func (m *Model) paletteKey(msg tea.KeyMsg) tea.Cmd {
	items := m.filteredCommands()
	switch msg.String() {
	case "up", "ctrl+k":
		m.palSel = maxInt(0, m.palSel-1)
	case "down", "ctrl+j":
		m.palSel = minInt(maxInt(0, len(items)-1), m.palSel+1)
	case "enter":
		m.overlay = ovNone
		if m.palSel < len(items) {
			return items[m.palSel].Do(m)
		}
	case "backspace":
		r := []rune(m.palQuery)
		if len(r) > 0 {
			m.palQuery = string(r[:len(r)-1])
		}
		m.palSel = 0
	default:
		if msg.Type == tea.KeyRunes {
			m.palQuery += string(msg.Runes)
			m.palSel = 0
		} else if msg.Type == tea.KeySpace {
			m.palQuery += " "
			m.palSel = 0
		}
	}
	return nil
}

// filteredCommands fuzzy-filters the registry by subsequence match.
func (m *Model) filteredCommands() []Command {
	all := commands()
	if m.palQuery == "" {
		return all
	}
	var out []Command
	q := strings.ToLower(m.palQuery)
	for _, c := range all {
		if fuzzyMatch(strings.ToLower(c.Name), q) {
			out = append(out, c)
		}
	}
	return out
}

func fuzzyMatch(s, q string) bool {
	i := 0
	for _, r := range s {
		if i < len(q) && rune(q[i]) == r {
			i++
		}
	}
	return i == len(q)
}

func (m *Model) colorKey(key string) tea.Cmd {
	n := len(Palette)
	switch key {
	case "left", "h", "up", "k":
		m.colorSel = (m.colorSel - 1 + n) % n
	case "right", "l", "down", "j", "tab":
		m.colorSel = (m.colorSel + 1) % n
	case "enter":
		if m.overlay == ovColorStroke {
			m.strokeIdx = m.colorSel
			idx := m.colorSel
			m.applyStyle("stroke color", func(o *scene.Object) { o.Stroke = Palette[idx] })
		} else {
			m.fillIdx = m.colorSel
			idx := m.colorSel
			m.applyStyle("fill color", func(o *scene.Object) {
				o.Fill = Palette[idx]
				o.Filled = true
			})
			m.filled = true
		}
		m.overlay = ovNone
	case "g":
		// Fill picker: apply the highlighted color as the gradient end color.
		if m.overlay == ovColorFill {
			idx := m.colorSel
			m.applyStyle("gradient", func(o *scene.Object) {
				c := Palette[idx]
				o.Fill2 = &c
				o.Filled = true
			})
			m.overlay = ovNone
			m.setStatus(statusOK, "gradient end color set (%s)", Palette[idx].Hex())
		}
	case "x":
		if m.overlay == ovColorFill {
			m.applyStyle("clear gradient", func(o *scene.Object) { o.Fill2 = nil })
			m.overlay = ovNone
			m.setStatus(statusInfo, "gradient cleared")
		}
	default:
		if len(key) == 1 && key[0] >= '1' && key[0] <= '9' && int(key[0]-'1') < n {
			m.colorSel = int(key[0] - '1')
		}
	}
	return nil
}

func (m *Model) layersKey(key string) tea.Cmd {
	d := m.doc
	switch key {
	case "up", "k":
		m.layerSel = maxInt(0, m.layerSel-1)
	case "down", "j":
		m.layerSel = minInt(len(d.Layers)-1, m.layerSel+1)
	case " ", "space", "v":
		m.checkpoint("layer visibility")
		d.Layers[m.layerSel].Visible = !d.Layers[m.layerSel].Visible
	case "x":
		m.checkpoint("layer lock")
		d.Layers[m.layerSel].Locked = !d.Layers[m.layerSel].Locked
	case "n":
		m.checkpoint("new layer")
		d.Layers = append(d.Layers, scene.Layer{Name: newLayerName(len(d.Layers)), Visible: true})
		m.layerSel = len(d.Layers) - 1
		m.setStatus(statusOK, "added %s", d.Layers[m.layerSel].Name)
	case "m":
		if len(m.sel) > 0 {
			m.checkpoint("move to layer")
			for _, o := range m.selection() {
				o.Layer = m.layerSel
			}
			m.setStatus(statusOK, "moved %d object(s) to %s", len(m.sel), d.Layers[m.layerSel].Name)
		}
	case "r":
		idx := m.layerSel
		m.overlay = ovNone
		m.prompt = &promptState{
			label: "Rename layer", value: d.Layers[idx].Name,
			confirm: func(m *Model, v string) tea.Cmd {
				if v != "" {
					m.checkpoint("rename layer")
					m.doc.Layers[idx].Name = v
				}
				return nil
			},
		}
	case "enter", "l", "L":
		m.overlay = ovNone
		m.setStatus(statusInfo, "current layer: %s", d.Layers[m.currentLayer()].Name)
	}
	return nil
}

func newLayerName(n int) string { return fmt.Sprintf("Layer %d", n+1) }
