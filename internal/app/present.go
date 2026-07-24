package app

import (
	"fmt"
	"math"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/anishhs-gh/canvux/internal/render"
	"github.com/anishhs-gh/canvux/internal/scene"
)

// Presentation mode (roadmap phase 14): each layer is a slide, rendered
// chrome-free and zoomed to fit. ←/→ navigate, esc exits.

func (m *Model) enterPresent() tea.Cmd {
	if len(m.doc.Layers) == 0 {
		return nil
	}
	m.present = true
	m.presentIdx = clampInt(m.currentLayer(), 0, len(m.doc.Layers)-1)
	m.overlay = ovNone
	m.prompt = nil
	return nil
}

func (m *Model) presentKey(key string) tea.Cmd {
	switch key {
	case "esc", "q", "P":
		m.present = false
	case "left", "up", "h", "k", "pgup":
		m.presentIdx = maxInt(0, m.presentIdx-1)
	case "right", "down", "l", "j", "space", " ", "enter", "pgdown":
		if m.presentIdx < len(m.doc.Layers)-1 {
			m.presentIdx++
		} else {
			m.present = false // walking off the last slide ends the show
		}
	case "ctrl+c":
		return tea.Quit
	}
	return nil
}

func (m *Model) presentMouse(msg tea.MouseMsg) tea.Cmd {
	if msg.Action != tea.MouseActionPress {
		return nil
	}
	switch msg.Button {
	case tea.MouseButtonLeft, tea.MouseButtonWheelDown:
		return m.presentKey("right")
	case tea.MouseButtonRight, tea.MouseButtonWheelUp:
		return m.presentKey("left")
	}
	return nil
}

// viewPresent renders the current slide full-screen.
func (m *Model) viewPresent() string {
	t := m.theme
	g := render.NewCellGrid(m.w, m.h, t.CanvasBG)
	g.Profile = m.profile
	sx, sy := m.mode.PixelScale()
	v := render.View{W: m.w * sx, H: m.h * sy, Zoom: 2}

	// Fit the slide's content.
	var objs []*scene.Object
	for _, o := range m.doc.Objects {
		if o.Layer == m.presentIdx {
			objs = append(objs, o)
		}
	}
	if len(objs) > 0 {
		b := objs[0].Bounds()
		for _, o := range objs[1:] {
			b = b.Union(o.Bounds())
		}
		b = b.Expand(3)
		v.Zoom = math.Min(float64(v.W)/math.Max(b.W(), 0.001), float64(v.H)/math.Max(b.H(), 0.001))
		v.Center = b.Center()
	}

	pb := render.NewPixelBuf(v.W, v.H)
	for _, o := range objs {
		render.DrawObject(pb, v, o)
	}
	g.Composite(pb, m.mode, 0, t.CanvasBG)
	for _, o := range objs {
		if o.Kind != scene.KindText {
			continue
		}
		p := v.ToPixel(o.P1)
		cx, cy := int(p.X)/sx, int(p.Y)/sy
		for i, r := range o.Text {
			under := g.Get(cx+i, cy)
			g.Set(cx+i, cy, render.Cell{Ch: r, Fg: o.Stroke, Bg: under.Bg})
		}
	}

	label := fmt.Sprintf(" %s · %d/%d · ←/→ · esc ",
		m.doc.Layers[m.presentIdx].Name, m.presentIdx+1, len(m.doc.Layers))
	g.SetString(m.w-len([]rune(label))-1, m.h-1, label, t.BarDim, t.CanvasBG)
	return g.ANSI()
}
