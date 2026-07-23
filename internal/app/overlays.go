package app

import (
	"fmt"

	"github.com/anishhs-gh/canvux/internal/render"
)

func (m *Model) drawOverlay(g *render.CellGrid) {
	switch m.overlay {
	case ovPalette:
		m.drawPalette(g)
	case ovHelp:
		m.drawHelp(g)
	case ovColorStroke, ovColorFill:
		m.drawColorPicker(g)
	case ovLayers:
		m.drawLayers(g)
	case ovStencils:
		m.drawStencils(g)
	}
}

func (m *Model) drawStencils(g *render.CellGrid) {
	t := m.theme
	items := m.filteredStencils()
	maxRows := minInt(14, m.h-8)
	w := minInt(48, m.w-4)
	h := maxRows + 4
	ix, iy := m.box(g, w, h, "Insert Stencil")
	inner := w - 4

	g.SetString(ix, iy, "› "+m.stencilQry, t.OverlayFG, t.OverlayBG)
	g.Set(ix+2+len([]rune(m.stencilQry)), iy, render.Cell{Ch: ' ', Fg: t.AccentText, Bg: t.Accent})
	iy++

	m.stencilSel = clampInt(m.stencilSel, 0, maxInt(0, len(items)-1))
	top := clampInt(m.stencilSel-maxRows+1, 0, maxInt(0, len(items)-maxRows))
	for i := 0; i < maxRows && top+i < len(items); i++ {
		it := items[top+i]
		fg, bg := t.OverlayFG, t.OverlayBG
		if top+i == m.stencilSel {
			fg, bg = t.OverlayFG, t.OverlaySel
			for x := ix - 1; x < ix-1+inner+2; x++ {
				g.Set(x, iy+1+i, render.Cell{Ch: ' ', Fg: fg, Bg: bg})
			}
		}
		g.SetString(ix, iy+1+i, it.Name, fg, bg)
		g.SetString(ix+inner-len([]rune(it.Cat)), iy+1+i, it.Cat, t.OverlayDim, bg)
	}
	if len(items) == 0 {
		g.SetString(ix, iy+1, "no matching stencils", t.OverlayDim, t.OverlayBG)
	}
}

// box draws a rounded bordered panel and returns its inner origin.
func (m *Model) box(g *render.CellGrid, w, h int, title string) (int, int) {
	t := m.theme
	w = minInt(w, m.w-2)
	h = minInt(h, m.h-2)
	x0 := (m.w - w) / 2
	y0 := (m.h - h) / 2
	g.FillRect(x0, y0, w, h, t.OverlayBG)
	for x := x0; x < x0+w; x++ {
		g.Set(x, y0, render.Cell{Ch: '─', Fg: t.BarDim, Bg: t.OverlayBG})
		g.Set(x, y0+h-1, render.Cell{Ch: '─', Fg: t.BarDim, Bg: t.OverlayBG})
	}
	for y := y0; y < y0+h; y++ {
		g.Set(x0, y, render.Cell{Ch: '│', Fg: t.BarDim, Bg: t.OverlayBG})
		g.Set(x0+w-1, y, render.Cell{Ch: '│', Fg: t.BarDim, Bg: t.OverlayBG})
	}
	g.Set(x0, y0, render.Cell{Ch: '╭', Fg: t.BarDim, Bg: t.OverlayBG})
	g.Set(x0+w-1, y0, render.Cell{Ch: '╮', Fg: t.BarDim, Bg: t.OverlayBG})
	g.Set(x0, y0+h-1, render.Cell{Ch: '╰', Fg: t.BarDim, Bg: t.OverlayBG})
	g.Set(x0+w-1, y0+h-1, render.Cell{Ch: '╯', Fg: t.BarDim, Bg: t.OverlayBG})
	if title != "" {
		g.SetString(x0+2, y0, " "+title+" ", t.Accent, t.OverlayBG)
	}
	return x0 + 2, y0 + 1
}

func (m *Model) drawPalette(g *render.CellGrid) {
	t := m.theme
	items := m.filteredCommands()
	maxRows := minInt(14, m.h-8)
	w := minInt(56, m.w-4)
	h := maxRows + 4
	ix, iy := m.box(g, w, h, "Command Palette")
	inner := w - 4

	g.SetString(ix, iy, "› "+m.palQuery, t.OverlayFG, t.OverlayBG)
	g.Set(ix+2+len([]rune(m.palQuery)), iy, render.Cell{Ch: ' ', Fg: t.AccentText, Bg: t.Accent})
	iy++

	m.palSel = clampInt(m.palSel, 0, maxInt(0, len(items)-1))
	top := clampInt(m.palSel-maxRows+1, 0, maxInt(0, len(items)-maxRows))
	for i := 0; i < maxRows && top+i < len(items); i++ {
		it := items[top+i]
		fg, bg := t.OverlayFG, t.OverlayBG
		if top+i == m.palSel {
			fg, bg = t.OverlayFG, t.OverlaySel
			for x := ix - 1; x < ix-1+inner+2; x++ {
				g.Set(x, iy+1+i, render.Cell{Ch: ' ', Fg: fg, Bg: bg})
			}
		}
		name := it.Name
		if len(name) > inner-12 {
			name = name[:inner-12] + "…"
		}
		g.SetString(ix, iy+1+i, name, fg, bg)
		if it.Keys != "" {
			g.SetString(ix+inner-len([]rune(it.Keys)), iy+1+i, it.Keys, t.OverlayDim, bg)
		}
	}
	if len(items) == 0 {
		g.SetString(ix, iy+1, "no matching commands", t.OverlayDim, t.OverlayBG)
	}
}

var helpLines = []string{
	"TOOLS",
	"  v select      b brush       l line        r rect",
	"  e ellipse     a arrow       p polygon     n curve",
	"  t text        x eraser      space pan (toggle)",
	"  i insert stencil (flowchart / UML / ER / arch / notes)",
	"",
	"CANVAS",
	"  wheel zoom at cursor        shift+wheel pan sideways",
	"  right/middle-drag pan       arrows/hjkl pan view",
	"  + - zoom     0 zoom 100%    F zoom to fit",
	"  g grid       G snap to grid M block/braille mode",
	"",
	"SELECT & EDIT",
	"  click select        shift+click multi-select",
	"  drag move           drag corner handles to resize",
	"  drag empty space marquee-select   tab cycle objects",
	"  arrows nudge selection (shift = fine)",
	"  d duplicate   Y copy   V paste   del delete",
	"  , . rotate ±15° (< > = ±1°)      [ ] z-order",
	"  ctrl+a select all   esc deselect / cancel",
	"",
	"STYLE",
	"  c stroke color      C fill color    1-9 quick stroke color",
	"  f toggle fill       D toggle dashed S toggle shadow",
	"  w/W stroke width +/-      O/o opacity +/-",
	"  B / ctrl+b blur +/-",
	"  gradient: C, pick color, press g (x clears)",
	"",
	"DRAWING",
	"  shift while drawing: square / circle / 45° lines",
	"  polygon: click vertices, enter or double-click to close",
	"  curve: drag, then drag its round control handles (select tool)",
	"  text: click, type, enter (double-click text to edit)",
	"",
	"FILES",
	"  ctrl+s save         ctrl+o open     ctrl+e export svg/png",
	"  autosaves every 30s to <file>.autosave when dirty",
	"",
	"UI",
	"  : or ctrl+p command palette    L layers    ? this help",
	"  u/ctrl+z undo   U/ctrl+y redo  q quit",
	"  P presentation mode (layers = slides, ←/→, esc)",
	"",
	"COLLABORATION",
	"  host:  canvux serve file.canvux    (then share host:7878)",
	"  join:  canvux join host:7878 --name you",
	"  peers' cursors appear live; edits merge last-writer-wins",
	"",
	"PLUGINS",
	"  executables named canvux-* in ~/.config/canvux/plugins",
	"  appear in the command palette; see canvux plugins",
}

func (m *Model) drawHelp(g *render.CellGrid) {
	t := m.theme
	h := minInt(len(helpLines)+4, m.h-2)
	w := minInt(64, m.w-4)
	ix, iy := m.box(g, w, h, "Canvux Help")
	rows := h - 3
	m.helpTop = clampInt(m.helpTop, 0, maxInt(0, len(helpLines)-rows))
	for i := 0; i < rows && m.helpTop+i < len(helpLines); i++ {
		line := helpLines[m.helpTop+i]
		fg := t.OverlayFG
		if line != "" && line[0] != ' ' {
			fg = t.Accent
		}
		if len(line) > w-4 {
			line = line[:w-4]
		}
		g.SetString(ix, iy+i, line, fg, t.OverlayBG)
	}
	if len(helpLines) > rows {
		g.SetString(ix, iy+rows, "↑/↓ scroll · esc close", t.OverlayDim, t.OverlayBG)
	}
}

func (m *Model) drawColorPicker(g *render.CellGrid) {
	t := m.theme
	title := "Stroke Color"
	if m.overlay == ovColorFill {
		title = "Fill Color"
	}
	w := minInt(len(Palette)*4+6, m.w-4)
	ix, iy := m.box(g, w, 6, title)
	for i, c := range Palette {
		x := ix + i*4
		g.SetString(x+1, iy+1, "██", c, t.OverlayBG)
		if i == m.colorSel {
			g.SetString(x, iy+1, "[", t.Accent, t.OverlayBG)
			g.SetString(x+3, iy+1, "]", t.Accent, t.OverlayBG)
		}
		g.SetString(x+1, iy+2, fmt.Sprintf("%d", (i+1)%10), t.OverlayDim, t.OverlayBG)
	}
	sel := Palette[m.colorSel]
	g.SetString(ix, iy+3, fmt.Sprintf("%s · ←/→ choose · 1-9 jump · enter apply", sel.Hex()), t.OverlayDim, t.OverlayBG)
}

func (m *Model) drawLayers(g *render.CellGrid) {
	t := m.theme
	d := m.doc
	h := minInt(len(d.Layers)+5, m.h-2)
	w := minInt(44, m.w-4)
	ix, iy := m.box(g, w, h, "Layers")
	m.layerSel = clampInt(m.layerSel, 0, len(d.Layers)-1)

	counts := make([]int, len(d.Layers))
	for _, o := range d.Objects {
		if o.Layer >= 0 && o.Layer < len(counts) {
			counts[o.Layer]++
		}
	}
	// Topmost layer first, like most editors.
	for row, i := 0, len(d.Layers)-1; i >= 0; i, row = i-1, row+1 {
		l := d.Layers[i]
		fg, bg := t.OverlayFG, t.OverlayBG
		if i == m.layerSel {
			bg = t.OverlaySel
			for x := ix - 1; x < ix+w-3; x++ {
				g.Set(x, iy+row, render.Cell{Ch: ' ', Fg: fg, Bg: bg})
			}
		}
		vis := "○"
		visFg := t.OverlayDim
		if l.Visible {
			vis, visFg = "●", t.OK
		}
		lock := " "
		if l.Locked {
			lock = "⊘"
		}
		cur := " "
		if i == m.currentLayer() {
			cur = "▸"
		}
		g.SetString(ix, iy+row, cur, t.Accent, bg)
		g.SetString(ix+2, iy+row, vis, visFg, bg)
		g.SetString(ix+4, iy+row, lock, t.Danger, bg)
		g.SetString(ix+6, iy+row, l.Name, fg, bg)
		cnt := fmt.Sprintf("%d obj", counts[i])
		g.SetString(ix+w-5-len(cnt), iy+row, cnt, t.OverlayDim, bg)
	}
	g.SetString(ix, iy+len(d.Layers)+1, "space show/hide · x lock · n new · r rename", t.OverlayDim, t.OverlayBG)
	g.SetString(ix, iy+len(d.Layers)+2, "m move sel here · enter set current · esc", t.OverlayDim, t.OverlayBG)
}
