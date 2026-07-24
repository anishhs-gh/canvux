package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/anishhs-gh/canvux/internal/geom"
	"github.com/anishhs-gh/canvux/internal/scene"
)

func addRect(m *Model, x1, y1, x2, y2 float64) *scene.Object {
	o := &scene.Object{
		Kind: scene.KindRect, P1: geom.V(x1, y1), P2: geom.V(x2, y2),
		Stroke: scene.Color{R: 255}, StrokeWidth: 1, Opacity: 1,
	}
	m.doc.Add(o)
	return o
}

// alignSnap must pull a moving box onto a nearby object's edge and emit a guide.
func TestAlignSnap(t *testing.T) {
	m := newTestModel(t)
	m.doc.Camera.Zoom = 2
	anchor := addRect(m, 0, 0, 10, 10) // left edge at x=0
	moving := geom.R(geom.V(0.3, 20), geom.V(10.3, 30))
	res := m.alignSnap(moving, map[uint64]bool{}, 1.0)
	if res.dx == 0 {
		t.Fatalf("expected x snap toward anchor left edge, got dx=0")
	}
	// Moving left edge 0.3 should snap to anchor left edge 0.0 → dx = -0.3.
	if res.dx > -0.29 || res.dx < -0.31 {
		t.Errorf("dx = %f, want ~-0.3", res.dx)
	}
	if len(res.guides) == 0 {
		t.Error("no guide emitted for the snap")
	}
	_ = anchor
}

func TestAlignSnapExcludesSelf(t *testing.T) {
	m := newTestModel(t)
	o := addRect(m, 0, 0, 10, 10)
	moving := o.Bounds()
	res := m.alignSnap(moving, map[uint64]bool{o.ID: true}, 1.0)
	if res.dx != 0 || res.dy != 0 || len(res.guides) != 0 {
		t.Errorf("object should not align to itself: %+v", res)
	}
}

// Text caret editing: insert in the middle, move across lines, delete.
func TestTextCaretEditing(t *testing.T) {
	m := newTestModel(t)
	m.textObj = &scene.Object{Kind: scene.KindText, StrokeWidth: 1, Opacity: 1}
	m.textCaret = 0
	typeStr := func(s string) {
		for _, r := range s {
			m.handleTextKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		}
	}
	press := func(k string) {
		// Build a KeyMsg whose String() matches k for the named keys we use.
		switch k {
		case "left":
			m.handleTextKey(tea.KeyMsg{Type: tea.KeyLeft})
		case "right":
			m.handleTextKey(tea.KeyMsg{Type: tea.KeyRight})
		case "up":
			m.handleTextKey(tea.KeyMsg{Type: tea.KeyUp})
		case "down":
			m.handleTextKey(tea.KeyMsg{Type: tea.KeyDown})
		case "home":
			m.handleTextKey(tea.KeyMsg{Type: tea.KeyHome})
		case "backspace":
			m.handleTextKey(tea.KeyMsg{Type: tea.KeyBackspace})
		case "newline":
			m.handleTextKey(tea.KeyMsg{Type: tea.KeyCtrlJ})
		}
	}
	typeStr("helloworld")
	// Move caret left 5 (before "world") and insert a space.
	for i := 0; i < 5; i++ {
		press("left")
	}
	typeStr(" ")
	if m.textObj.Text != "hello world" {
		t.Fatalf("mid insert failed: %q", m.textObj.Text)
	}
	// Newline then a second line. The caret sits before "world" (right after
	// the inserted space), so the newline splits there and "line2" lands at the
	// start of the new line, ahead of "world".
	press("newline")
	typeStr("line2")
	if m.textObj.Text != "hello \nline2world" {
		t.Fatalf("newline/insert wrong: %q", m.textObj.Text)
	}
	// Caret up then home, then backspace at line start is a no-op deleting nothing.
	press("home")
	before := m.textObj.Text
	// Move up to the first line, to its start, delete one char back (removes newline join point? no—at col0 line0 there's nothing before)
	press("up")
	press("home")
	press("backspace")
	if m.textObj.Text != before {
		t.Errorf("backspace at document start changed text: %q -> %q", before, m.textObj.Text)
	}
}

func TestMultiLineTextBounds(t *testing.T) {
	o := &scene.Object{Kind: scene.KindText, P1: geom.V(0, 0), Text: "ab\ncdef\ng"}
	b := o.Bounds()
	if b.W() < 4 {
		t.Errorf("width should cover longest line (4): got %f", b.W())
	}
	if b.H() < 4 { // 3 lines * 2 nominal - allow >=
		t.Errorf("height should grow with lines: got %f", b.H())
	}
}

func TestTextLines(t *testing.T) {
	if got := scene.TextLines(""); len(got) != 1 || got[0] != "" {
		t.Errorf("empty text = %v, want one empty line", got)
	}
	if got := scene.TextLines("a\nb\nc"); len(got) != 3 {
		t.Errorf("3-line split = %v", got)
	}
}

// Keyboard-draw: place a rectangle with two enter presses, no mouse.
func TestKeyboardDraw(t *testing.T) {
	m := newTestModel(t)
	m.w, m.h = 100, 30
	m.doc.Camera.Zoom = 2
	m.setTool(ToolRect)
	m.toggleKbDraw()
	if !m.kbDraw {
		t.Fatal("kbDraw did not enable")
	}
	// Place start, move cursor, place end.
	m.kbPlacePoint()
	if m.draft == nil {
		t.Fatal("first placement should start a draft")
	}
	m.kbCursor = m.kbCursor.Add(geom.V(20, 12))
	m.updateKbDraft()
	m.kbPlacePoint()
	if m.draft != nil {
		t.Error("second placement should commit the draft")
	}
	rects := 0
	for _, o := range m.doc.Objects {
		if o.Kind == scene.KindRect {
			rects++
		}
	}
	if rects != 1 {
		t.Fatalf("keyboard draw produced %d rects, want 1", rects)
	}
}
