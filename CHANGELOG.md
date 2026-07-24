# Changelog

All notable changes to Canvux are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/) and versions follow
[Semantic Versioning](https://semver.org/).

## [Unreleased]

_Nothing yet._

## [1.0.0] — 2026-07-24

Initial release. A terminal-native infinite vector canvas in a single static
Go binary — draw with mouse and keyboard, collaborate in real time, and export
standards-compliant SVG and PNG.

### Canvas & navigation
- Infinite world with pan and zoom (0.05×–400×), wheel-zoom at the cursor,
  right/middle-drag pan, and an adaptive dot grid
- Two render modes: colorful half-block cells, or hi-res braille (2×4 subpixels
  per cell), toggled with `M`
- Runs on any terminal: truecolor by default, auto-degrading to 256-color,
  16-color, or monochrome (honors `NO_COLOR`; override with `--color`)

### Tools
- 11 tools: select, pan, brush, line, rectangle, ellipse, arrow, polygon,
  bézier curve, text, eraser
- Bézier curves with draggable control handles
- Speed-sensitive variable-width brush; freehand strokes are smoothed and
  simplified on release
- Multi-line text with a full caret (arrows, Home/End, `ctrl+j` for newlines)

### Editing
- Move, corner-handle resize, rotate, duplicate, multi-select, marquee select
- Undo/redo (200 levels), copy/paste, z-order, layers with hide/lock
- Snap to grid and **smart alignment guides** that snap to other objects' edges
  and centers
- Advanced styling: linear gradients, drop shadows, blur, dashed strokes,
  opacity, rotation

### Diagramming
- Insert-stencil library (`i`): 19 prefabs across flowchart, UML, ER,
  architecture, and misc (sticky note, table, mind map)

### Files & interop
- Git-friendly `.canvux` documents (indented JSON, versioned schema)
- Autosave with crash recovery prompt on reopen
- SVG export (gradients, filters, bézier paths) and import (shapes + paths)
- PNG export at any scale; embed PNG/JPEG/GIF images as run-merged vector rects
- `canvux diff` for object-level document diffs
- Copy the selection as SVG to the system clipboard (`ctrl+shift+c`; native
  tools locally, OSC 52 over SSH)

### Collaboration
- `canvux serve` hosts a document; others `canvux join` and edit together with
  live peer cursors and last-writer-wins merge

### Extensibility
- Plugin system: any executable named `canvux-*` extends the editor via JSON
  over stdin/stdout (generators, importers, exporters, transforms)
- CLI: `render`, `export`, `add`, `diff`, `info`, `serve`, `join`, `plugins`

### Accessibility & UX
- Local cursor crosshair, live `W×H` / length∠angle readouts while drawing,
  animated marching-ants selection
- Configurable via `~/.config/canvux/config.json` (project `./.canvux.json`
  overrides): theme, palette, render mode, grid/snap, autosave, and a fully
  rebindable keymap
- Colorblind-safe (Okabe–Ito) and high-contrast palettes/themes
- Outline navigator (`: outline`): a textual, keyboard-first object list
- Keyboard-only drawing mode (`K`) for mouse-free shape creation
- Presentation mode (`P`) turns layers into slides; light "whiteboard" theme

### Performance
- In-place render buffer reuse: ~0 allocations per frame
- ~5 ms to rasterize a 10,000-object frame; single ~4.5 MB binary, no runtime
  dependencies

[Unreleased]: https://github.com/anishhs-gh/canvux/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/anishhs-gh/canvux/releases/tag/v1.0.0
