<div align="center">

# ◆ Canvux

**A terminal-native infinite vector canvas — in a single static binary.**

Draw with mouse *and* keyboard, edit real vector objects, collaborate in real
time, and export standards-compliant SVG and PNG. No browser, no Electron, no
cloud.

[![CI](https://github.com/anishhs-gh/canvux/actions/workflows/ci.yml/badge.svg)](https://github.com/anishhs-gh/canvux/actions/workflows/ci.yml)
[![Audit](https://github.com/anishhs-gh/canvux/actions/workflows/audit.yml/badge.svg)](https://github.com/anishhs-gh/canvux/actions/workflows/audit.yml)
[![Release](https://img.shields.io/github/v/release/anishhs-gh/canvux?sort=semver)](https://github.com/anishhs-gh/canvux/releases)
[![Go](https://img.shields.io/github/go-mod/go-version/anishhs-gh/canvux)](go.mod)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

![Canvux demo — a diagram drawn in Canvux and exported to PNG](examples/demo.png)

<sub>Drawn in Canvux, exported with `canvux export` — the same rasterizer drives the terminal view.</sub>

</div>

---

## Contents

- [Highlights](#highlights)
- [Install](#install)
- [Quick start](#quick-start)
- [Command-line usage](#command-line-usage)
- [Configuration](#configuration)
- [Collaboration](#collaboration)
- [Plugins](#plugins)
- [Architecture](#architecture)
- [Development](#development)
- [Author](#author)
- [License](#license)

## Highlights

**Canvas & navigation**
- Infinite world, pan and zoom (0.05×–400×), wheel-zoom at the cursor, adaptive grid
- Half-block or hi-res braille rendering (`M`) — works on any terminal, truecolor down to monochrome (`--color`, honors `NO_COLOR`)

**Tools & editing**
- 11 tools: select, pan, brush, line, rect, ellipse, arrow, polygon, **bézier**, text, eraser
- Move / resize / rotate / duplicate / multi-select, undo–redo (200 levels), z-order, layers
- **Smart alignment guides** snap to other objects' edges and centers; snap-to-grid
- Gradients, drop shadows, blur, dashed strokes, opacity, variable-width brush with auto-smoothing
- Multi-line text with a full caret

**Beyond freehand**
- **Diagram stencils** (`i`): flowchart, UML, ER, architecture, sticky notes, tables, mind maps
- **Realtime collaboration** with live peer cursors (`canvux serve` / `canvux join`)
- **Plugins**: any `canvux-*` executable extends the editor via JSON over stdio
- **Presentation mode** (`P`), light "whiteboard" theme, image embedding

**Files & interop**
- Git-friendly `.canvux` JSON, autosave with crash recovery, `canvux diff`
- SVG export/import, PNG export, copy-as-SVG to the system clipboard

**Accessibility**
- Colorblind-safe and high-contrast palettes/themes
- Textual **Outline navigator** (`: outline`) and **keyboard-only drawing** (`K`)
- Fully rebindable keymap

**Fast & small** — ~5 ms to rasterize 10,000 objects, ~0 allocations per frame, a single ~4.5 MB binary with no runtime dependencies.

## Install

**One-liner** (downloads a prebuilt binary, or builds from source as a last resort):

```sh
curl -fsSL https://raw.githubusercontent.com/anishhs-gh/canvux/main/install.sh | sh
```

**With Go:**

```sh
go install github.com/anishhs-gh/canvux/cmd/canvux@latest
```

**From source:**

```sh
git clone git@github.com:anishhs-gh/canvux.git
cd canvux && make build   # -> ./canvux
```

To remove: `./uninstall.sh` (add `--purge` to also drop `~/.config/canvux`).

## Quick start

```sh
canvux                       # blank canvas
canvux drawing.canvux        # open (created on first save)
```

Press <kbd>?</kbd> for the full in-app help. The 60-second tour:

| Do this | To |
|---|---|
| <kbd>r</kbd> then drag | draw a rectangle (<kbd>shift</kbd> = square) |
| <kbd>f</kbd> · <kbd>c</kbd> / <kbd>C</kbd> | toggle fill · pick stroke / fill color |
| <kbd>v</kbd> | select — drag to move, corner handles resize, <kbd>del</kbd> deletes |
| <kbd>t</kbd>, click, type | text (<kbd>ctrl</kbd>+<kbd>j</kbd> for a newline) |
| <kbd>n</kbd> then drag | bézier curve — reselect and drag its round handles |
| <kbd>i</kbd> | insert a diagram stencil (search "class", "cloud", …) |
| <kbd>K</kbd> | keyboard-only drawing (no mouse) |
| wheel / right-drag | zoom / pan |
| <kbd>u</kbd> / <kbd>U</kbd> | undo / redo |
| <kbd>:</kbd> | command palette · <kbd>L</kbd> layers · <kbd>P</kbd> present |
| <kbd>ctrl</kbd>+<kbd>s</kbd> / <kbd>ctrl</kbd>+<kbd>e</kbd> | save / export |

## Command-line usage

```sh
canvux [file] [--color auto|truecolor|256|16|off]   # open the editor
canvux render  drawing.canvux [--braille]           # print a drawing to stdout
canvux export  drawing.canvux --svg out.svg --png out.png --scale 10
canvux add     rect drawing.canvux --at 0,0 --size 20,10 --fill '#334'
canvux diff    old.canvux new.canvux                # object-level diff (exit 1 if differ)
canvux info    drawing.canvux                       # document stats
canvux serve   team.canvux                          # host a collaboration session
canvux join    host:7878 --name alice               # join one
canvux plugins                                      # list discovered plugins
canvux version
```

## Configuration

Canvux reads `~/.config/canvux/config.json`, then `./.canvux.json` in the
current directory (project overrides global). Every field is optional — see
[`examples/config.example.json`](examples/config.example.json):

```json
{
  "theme": "high-contrast",      // dark | light | high-contrast
  "palette": "colorblind",       // default | colorblind | high-contrast
  "renderMode": "braille",       // block | braille
  "color": "auto",               // auto | truecolor | 256 | 16 | off
  "grid": true,
  "snap": false,
  "autosaveSeconds": 30,
  "keys": { "ctrl+d": "edit.duplicate", "T": "view.cycle-theme" }
}
```

Any key can be remapped to any action ID (the command palette shows each
command's current binding). Theme and palette also cycle live from the palette.

## Collaboration

```sh
# on the host
canvux serve team.canvux                    # listens on :7878

# on each collaborator
canvux join 192.168.1.5:7878 --name alice
```

An authoritative server owns the document; clients exchange object-level
operations over TCP and see each other's live cursors. Edits merge
last-writer-wins; object IDs are namespaced per client so concurrent creation
never collides. The server autosaves and persists on shutdown.

## Plugins

A plugin is any executable named `canvux-*` on the plugin path, speaking JSON
over stdin/stdout — so you can write one in any language. Commands appear in the
command palette. See [`examples/plugins/`](examples/plugins/) for the protocol
and a working flowchart generator.

## Architecture

```
cmd/canvux         CLI entrypoint (editor + subcommands)
internal/app       interactive editor: tools, input, overlays, UI, keymap
internal/scene     scene graph: objects, layers, hit-testing, serialization
internal/render    rasterizer (Surface interface) + compositors + color profiles
internal/svg       SVG export + import
internal/export    PNG export
internal/imgembed  image → run-merged vector rects
internal/config    layered JSON configuration
internal/plugin    external-process plugin protocol
internal/collab    realtime collaboration server + client
internal/clipboard system clipboard (native tools + OSC 52)
internal/history   snapshot undo/redo
internal/geom      2D vector math
```

The rasterizer draws into an abstract `render.Surface`, so the terminal buffer
and the PNG exporter share the same drawing code.

## Development

```sh
make test              # go vet + go test ./...
make bench             # render benchmarks
go test -race ./...    # what CI enforces
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for setup, conventions, and the PR flow,
and [CHANGELOG.md](CHANGELOG.md) for release history.

## Author

**Anish Shekh**
[GitHub @anishhs-gh](https://github.com/anishhs-gh) ·
[LinkedIn @anishsh](https://www.linkedin.com/in/anishsh) ·
[anishhs.com](https://anishhs.com)

## License

[MIT](LICENSE) © Anish Shekh
