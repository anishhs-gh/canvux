package app

import (
	"testing"

	"github.com/anishhs-gh/canvux/internal/config"
	"github.com/anishhs-gh/canvux/internal/scene"
)

// newTestModel builds a model without touching the filesystem (no config load,
// no plugin discovery), so tests are hermetic.
func newTestModel(t *testing.T) *Model {
	t.Helper()
	return &Model{
		doc:          scene.NewDoc(),
		theme:        DefaultTheme,
		keymap:       DefaultKeymap(),
		sel:          map[uint64]bool{},
		strokeIdx:    1,
		fillIdx:      3,
		strokeWidth:  1,
		opacity:      1,
		autosaveSecs: 30,
	}
}

// Every key in the default keymap must point at a real action.
func TestDefaultKeymapResolves(t *testing.T) {
	for key, id := range DefaultKeymap() {
		if _, ok := actionByID(id); !ok {
			t.Errorf("key %q bound to unknown action %q", key, id)
		}
	}
}

// Action IDs must be unique and non-empty (the keymap and config rely on them).
func TestActionIDsUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, a := range actionTable() {
		if a.ID == "" || a.Name == "" || a.Do == nil {
			t.Errorf("malformed action: %+v", a)
		}
		if seen[a.ID] {
			t.Errorf("duplicate action ID %q", a.ID)
		}
		seen[a.ID] = true
	}
}

// The palette (m.commands) must expose every action with the correct key hint.
func TestPaletteReflectsKeymap(t *testing.T) {
	m := newTestModel(t)
	byName := map[string]Command{}
	for _, c := range m.commands() {
		byName[c.Name] = c
	}
	// "Tool: Rectangle" is bound to r by default.
	if c, ok := byName["Tool: Rectangle"]; !ok || c.Keys != "r" {
		t.Errorf("rectangle palette hint = %q, want r", c.Keys)
	}
	// After a rebind, the palette hint must follow.
	m.keymap["ctrl+shift+r"] = "tool.rect"
	delete(m.keymap, "r")
	for _, c := range m.commands() {
		if c.Name == "Tool: Rectangle" && c.Keys != "ctrl+shift+r" {
			t.Errorf("rebind not reflected in palette: %q", c.Keys)
		}
	}
}

// applyConfig must honor theme, palette, keymap overrides and flags.
func TestApplyConfig(t *testing.T) {
	m := newTestModel(t)
	no := false
	secs := 5
	m.applyConfig(config.Config{
		Theme: "high-contrast", Palette: "colorblind", RenderMode: "braille",
		Grid: &no, Autosave: &secs,
		Keys: map[string]string{"z": "edit.undo", "bogus": "does.not.exist"},
	})
	if m.theme.CanvasBG != HighContrastTheme.CanvasBG {
		t.Error("theme not applied")
	}
	if m.paletteIdx != 1 || &Palette[0] == &defaultPalette[0] {
		t.Error("palette not switched to colorblind")
	}
	if m.showGrid {
		t.Error("grid=false not applied")
	}
	if m.autosaveSecs != 5 {
		t.Error("autosave not applied")
	}
	if m.keymap["z"] != "edit.undo" {
		t.Error("valid key override not applied")
	}
	if _, ok := m.keymap["bogus"]; ok {
		t.Error("override to unknown action should be dropped")
	}
	SetActivePalette("default") // restore global for other tests
}

func TestThemeAndPaletteRegistries(t *testing.T) {
	if _, ok := ThemeByName("high-contrast"); !ok {
		t.Error("high-contrast theme missing")
	}
	if _, ok := ThemeByName("nope"); ok {
		t.Error("unknown theme should not resolve")
	}
	for _, p := range Palettes {
		if len(p.Colors) < 8 {
			t.Errorf("palette %q has too few colors: %d", p.Name, len(p.Colors))
		}
	}
	if !SetActivePalette("colorblind") {
		t.Error("SetActivePalette(colorblind) failed")
	}
	SetActivePalette("default")
}
