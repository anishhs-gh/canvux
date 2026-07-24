package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingIsEmpty(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CANVUX_CONFIG", filepath.Join(dir, "nope.json"))
	// Run from an empty dir so no project-local file exists either.
	t.Chdir(dir)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Theme != "" || cfg.Keys != nil || len(cfg.Sources) != 0 {
		t.Errorf("missing config should be empty, got %+v", cfg)
	}
}

func TestProjectOverridesGlobal(t *testing.T) {
	dir := t.TempDir()
	global := filepath.Join(dir, "global.json")
	os.WriteFile(global, []byte(`{
		"theme": "dark", "palette": "default", "autosaveSeconds": 60,
		"keys": {"z": "edit.undo", "y": "edit.redo"}
	}`), 0o644)
	t.Setenv("CANVUX_CONFIG", global)

	proj := t.TempDir()
	t.Chdir(proj)
	os.WriteFile(ProjectPath, []byte(`{
		"theme": "high-contrast", "grid": false,
		"keys": {"y": "edit.paste"}
	}`), 0o644)

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Theme != "high-contrast" {
		t.Errorf("project should override theme: got %q", cfg.Theme)
	}
	if cfg.Palette != "default" {
		t.Errorf("global palette should survive: got %q", cfg.Palette)
	}
	if cfg.Grid == nil || *cfg.Grid != false {
		t.Errorf("project grid=false not applied: %v", cfg.Grid)
	}
	if cfg.Autosave == nil || *cfg.Autosave != 60 {
		t.Errorf("global autosave should survive: %v", cfg.Autosave)
	}
	// Key maps merge, with project winning on conflicts.
	if cfg.Keys["z"] != "edit.undo" {
		t.Errorf("global key z lost: %q", cfg.Keys["z"])
	}
	if cfg.Keys["y"] != "edit.paste" {
		t.Errorf("project key y should override global: %q", cfg.Keys["y"])
	}
	if len(cfg.Sources) != 2 {
		t.Errorf("expected 2 sources, got %v", cfg.Sources)
	}
}

func TestMalformedConfigErrors(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.json")
	os.WriteFile(bad, []byte(`{ not json `), 0o644)
	t.Setenv("CANVUX_CONFIG", bad)
	t.Chdir(t.TempDir())
	if _, err := Load(); err == nil {
		t.Fatal("expected error for malformed config")
	}
}
