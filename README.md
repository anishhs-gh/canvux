# Canvux

**A terminal-native infinite vector canvas.** Draw with mouse and keyboard, edit vector objects, organize layers, and export standards-compliant SVG or PNG — all from a single static Go binary.

![Canvux demo export](examples/demo.png)

*The diagram above was drawn in Canvux and exported with `canvux export`.*

## Highlights

- **Infinite canvas** — pan and zoom (0.05x–400x) over unbounded world coordinates
- **Full mouse support** — draw, drag-move, marquee-select, corner-handle resize, wheel zoom at cursor, clickable toolbar
- **10 tools** — select, pan, brush, line, rect, ellipse, arrow, polygon, text, eraser
- **Two render modes** — colorful half-block cells, or hi-res braille (2×4 dots per cell), toggle with `M`
- **Real editor features** — undo/redo (200 levels), copy/paste, duplicate, rotate, z-order, snap-to-grid, shift-constrained drawing, multi-select, layers with lock/hide
- **Command palette** — `:` fuzzy-searches every command
- **Git-friendly files** — `.canvux` is indented JSON with a versioned schema; autosaves every 30 s
- **SVG round-trip** — export clean SVG, import basic SVG shapes back
- **PNG export** — rasterized at any scale from the same renderer
- **Scriptable CLI** — `render`, `export`, `info` subcommands for pipelines
- **Fast** — a frame with 10 000 objects rasterizes in ~5 ms; single ~4 MB binary, no runtime deps

## Install

```sh
go install github.com/anishhs-gh/canvux/cmd/canvux@latest
# or from a checkout:
make build      # -> ./canvux
```

## Use

```sh
canvux                       # blank canvas
canvux drawing.canvux        # open (or create on save) a project
canvux render examples/demo.canvux          # print a drawing to stdout
canvux export drawing.canvux --svg out.svg --png out.png --scale 10
canvux info drawing.canvux   # stats
```

### The 90-second tour

| | |
|---|---|
| `r` then drag | draw a rectangle (`shift` = square) |
| `f` | toggle fill · `c` / `C` pick stroke / fill color |
| `v` | select — drag to move, corner handles resize, `del` deletes |
| `t`, click, type | place text (double-click text to edit) |
| `p`, click points, `enter` | polygon |
| wheel / right-drag | zoom at cursor / pan |
| `u` / `U` | undo / redo |
| `:` | command palette · `?` full help · `L` layers |
| `ctrl+s` / `ctrl+e` | save / export |

Everything else is in the in-app help (`?`).

## File format

`.canvux` files are indented JSON: `version`, `camera`, `layers`, `objects`, `metadata`. Forward-incompatible files are rejected by version check; missing style fields get sane defaults, so the format can grow. See [`examples/demo.canvux`](examples/demo.canvux).

## Architecture

```
cmd/canvux        CLI: editor, render, export, info
internal/app      bubbletea model: tools, input, overlays, toolbar/status UI
internal/scene    scene graph: objects, layers, hit-testing, (de)serialization
internal/render   rasterizer (Surface interface) + cell compositors (half-block/braille) + ANSI
internal/svg      SVG export + import
internal/export   PNG export (image-backed Surface, micro bitmap font)
internal/history  snapshot undo/redo
internal/geom     vectors, rects, transforms
```

The rasterizer draws into an abstract `Surface`, so the terminal buffer and the PNG exporter share the same drawing code.

## Development

```sh
make test        # unit tests (JSON/SVG round-trip, hit testing, rendering, culling)
make bench       # 10k-object frame benchmark
```

## Roadmap status

Phases 0–8 of [the roadmap](canvux-roadmap.md) are implemented (foundation, camera, rendering, scene graph, editing, drawing tools, project files, SVG, productivity), plus parts of 9 (dashed strokes, opacity, rotation) and 12 (CLI scripting). Next up: Bézier curves, gradients, diagram libraries, plugins.

## License

MIT
