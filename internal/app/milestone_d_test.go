package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/anishhs-gh/canvux/internal/geom"
	"github.com/anishhs-gh/canvux/internal/scene"
)

func writeDoc(t *testing.T, path string, objs int) {
	t.Helper()
	d := scene.NewDoc()
	for i := 0; i < objs; i++ {
		d.Add(&scene.Object{Kind: scene.KindRect, P1: geom.V(0, 0), P2: geom.V(1, 1),
			Stroke: scene.Color{R: 255}, StrokeWidth: 1, Opacity: 1})
	}
	if err := d.Save(path); err != nil {
		t.Fatal(err)
	}
}

func TestAutosaveRecoveryPromptsWhenNewer(t *testing.T) {
	dir := t.TempDir()
	main := filepath.Join(dir, "d.canvux")
	writeDoc(t, main, 1)
	// Autosave with more objects, made newer than the main file.
	writeDoc(t, main+".autosave", 5)
	future := time.Now().Add(time.Minute)
	os.Chtimes(main+".autosave", future, future)

	m := newTestModel(t)
	m.path = main
	m.checkAutosaveRecovery()
	if m.prompt == nil || !m.prompt.yesNo {
		t.Fatal("expected a recovery y/n prompt")
	}
	// Accepting recovery loads the autosave (5 objects) and marks dirty.
	m.prompt.onYes(m)
	if len(m.doc.Objects) != 5 {
		t.Errorf("recovery loaded %d objects, want 5", len(m.doc.Objects))
	}
	if !m.dirty {
		t.Error("recovered doc should be dirty until saved")
	}
}

func TestNoRecoveryWhenAutosaveOlder(t *testing.T) {
	dir := t.TempDir()
	main := filepath.Join(dir, "d.canvux")
	writeDoc(t, main+".autosave", 5)
	// Main file written after the autosave → autosave is stale, no prompt.
	time.Sleep(10 * time.Millisecond)
	writeDoc(t, main, 1)

	m := newTestModel(t)
	m.path = main
	m.checkAutosaveRecovery()
	if m.prompt != nil {
		t.Error("stale autosave should not prompt recovery")
	}
}

func TestNoRecoveryWithoutAutosave(t *testing.T) {
	dir := t.TempDir()
	main := filepath.Join(dir, "d.canvux")
	writeDoc(t, main, 1)
	m := newTestModel(t)
	m.path = main
	m.checkAutosaveRecovery()
	if m.prompt != nil {
		t.Error("no autosave should not prompt")
	}
}

func TestCopySVGSetsOSCWhenNoNativeTool(t *testing.T) {
	m := newTestModel(t)
	m.w, m.h = 80, 24
	o := &scene.Object{Kind: scene.KindRect, P1: geom.V(0, 0), P2: geom.V(10, 10),
		Stroke: scene.Color{R: 255}, StrokeWidth: 1, Opacity: 1}
	m.doc.Add(o)
	m.sel[o.ID] = true

	// Force the no-native-tool path by clearing PATH so lookups fail.
	t.Setenv("PATH", "")
	cmd := m.cmdCopySVG()
	if cmd != nil {
		// A native tool was somehow still found; the async path is also valid.
		return
	}
	if m.pendingOSC == "" {
		t.Fatal("expected OSC 52 fallback to be queued")
	}
	if m.pendingOSC[:5] != "\x1b]52;"[:5] {
		t.Errorf("pendingOSC is not an OSC 52 sequence: %q", m.pendingOSC[:8])
	}
}

func TestWindowTitle(t *testing.T) {
	m := newTestModel(t)
	if got := m.windowTitle(); got != "Canvux — untitled" {
		t.Errorf("untitled title = %q", got)
	}
	m.path = "drawing.canvux"
	if got := m.windowTitle(); got != "Canvux — drawing.canvux" {
		t.Errorf("named title = %q", got)
	}
}
