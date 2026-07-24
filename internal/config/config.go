// Package config loads Canvux user preferences from JSON, merging a global
// file with a project-local override so teams can pin per-project defaults.
//
// Precedence (later wins): built-in defaults → ~/.config/canvux/config.json →
// ./.canvux.json (or $CANVUX_CONFIG if set, which replaces the global path).
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config is the user-tunable settings surface. Pointer/empty-string fields
// distinguish "unset" (inherit) from an explicit value during merge.
type Config struct {
	Theme      string            `json:"theme,omitempty"`      // dark | light | high-contrast
	Palette    string            `json:"palette,omitempty"`    // default | colorblind | high-contrast
	RenderMode string            `json:"renderMode,omitempty"` // block | braille
	Color      string            `json:"color,omitempty"`      // auto | truecolor | 256 | 16 | off
	Grid       *bool             `json:"grid,omitempty"`
	Snap       *bool             `json:"snap,omitempty"`
	Autosave   *int              `json:"autosaveSeconds,omitempty"`
	Keys       map[string]string `json:"keys,omitempty"` // key string -> action ID overrides

	// Sources lists the files that contributed, for diagnostics.
	Sources []string `json:"-"`
}

// GlobalPath returns the global config file path ($CANVUX_CONFIG wins).
func GlobalPath() string {
	if p := os.Getenv("CANVUX_CONFIG"); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "canvux", "config.json")
}

// ProjectPath is the project-local override in the current directory.
const ProjectPath = ".canvux.json"

// Load reads and merges the global then project-local config. Missing files
// are not errors; malformed files return an error naming the file.
func Load() (Config, error) {
	var cfg Config
	for _, path := range []string{GlobalPath(), ProjectPath} {
		if path == "" {
			continue
		}
		c, ok, err := loadFile(path)
		if err != nil {
			return cfg, err
		}
		if ok {
			cfg.merge(c)
			cfg.Sources = append(cfg.Sources, path)
		}
	}
	return cfg, nil
}

func loadFile(path string) (Config, bool, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Config{}, false, nil
	}
	if err != nil {
		return Config{}, false, err
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return Config{}, false, fmt.Errorf("parse %s: %w", path, err)
	}
	return c, true, nil
}

// merge overlays non-empty fields of o onto c.
func (c *Config) merge(o Config) {
	if o.Theme != "" {
		c.Theme = o.Theme
	}
	if o.Palette != "" {
		c.Palette = o.Palette
	}
	if o.RenderMode != "" {
		c.RenderMode = o.RenderMode
	}
	if o.Color != "" {
		c.Color = o.Color
	}
	if o.Grid != nil {
		c.Grid = o.Grid
	}
	if o.Snap != nil {
		c.Snap = o.Snap
	}
	if o.Autosave != nil {
		c.Autosave = o.Autosave
	}
	if o.Keys != nil {
		if c.Keys == nil {
			c.Keys = map[string]string{}
		}
		for k, v := range o.Keys {
			c.Keys[k] = v
		}
	}
}
