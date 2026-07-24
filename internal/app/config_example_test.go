package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/anishhs-gh/canvux/internal/config"
)

// The shipped example config must parse and reference only real theme,
// palette, and action names — so it never drifts from the code.
func TestExampleConfigIsValid(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "config.example.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read example config: %v", err)
	}
	var cfg config.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("example config does not parse: %v", err)
	}
	if _, ok := ThemeByName(cfg.Theme); !ok {
		t.Errorf("example theme %q is not a real theme", cfg.Theme)
	}
	if cfg.Palette != "" && !paletteExists(cfg.Palette) {
		t.Errorf("example palette %q is not a real palette", cfg.Palette)
	}
	for key, id := range cfg.Keys {
		if _, ok := actionByID(id); !ok {
			t.Errorf("example key %q → unknown action %q", key, id)
		}
	}
	// Applying it must not panic and must take effect.
	m := newTestModel(t)
	m.applyConfig(cfg)
	SetActivePalette("default") // restore global
}

func paletteExists(name string) bool {
	for _, p := range Palettes {
		if p.Name == name {
			return true
		}
	}
	return false
}
