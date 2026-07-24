package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/anishhs-gh/canvux/internal/clipboard"
	"github.com/anishhs-gh/canvux/internal/scene"
	"github.com/anishhs-gh/canvux/internal/svg"
)

// clipboardResult reports the outcome of an async native clipboard copy.
type clipboardResult struct {
	n   int
	err error
}

// cmdCopySVG copies the current selection (or the whole document if nothing is
// selected) to the system clipboard as an SVG snippet, so it can be pasted
// into other tools. Prefers a native clipboard tool; falls back to OSC 52 for
// SSH sessions.
func (m *Model) cmdCopySVG() tea.Cmd {
	objs := m.selection()
	if len(objs) == 0 {
		objs = m.doc.VisibleObjects()
	}
	if len(objs) == 0 {
		m.setStatus(statusInfo, "nothing to copy")
		return nil
	}
	sub := scene.NewDoc()
	sub.Layers = m.doc.Layers
	for _, o := range objs {
		sub.Objects = append(sub.Objects, o)
	}
	data := string(svg.Export(sub))
	n := len(objs)

	if clipboard.HasNative() {
		return func() tea.Msg {
			err := clipboard.Native(data)
			return clipboardResult{n: n, err: err}
		}
	}
	// No native tool (likely SSH): emit OSC 52 with the next frame.
	m.pendingOSC = clipboard.OSC52(data)
	m.setStatus(statusOK, "copied %d object(s) as SVG (OSC 52)", n)
	return nil
}

// handleClipboardResult surfaces the native-copy outcome.
func (m *Model) handleClipboardResult(r clipboardResult) {
	if r.err != nil {
		m.setStatus(statusErr, "clipboard: %v", r.err)
		return
	}
	m.setStatus(statusOK, "copied %d object(s) as SVG to clipboard", r.n)
}
