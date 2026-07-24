# Contributing to Canvux

Thanks for your interest in improving Canvux! This guide covers how to get set
up, the conventions the project follows, and how changes get merged.

## Getting started

You need **Go** (the version is pinned in [`go.mod`](go.mod)). No other runtime
dependencies.

```sh
git clone git@github.com:anishhs-gh/canvux.git
cd canvux
make build      # -> ./canvux
./canvux examples/demo.canvux
```

## Development workflow

```sh
make test       # go vet + go test ./...
make bench      # render benchmarks
go test -race ./...   # what CI runs
```

Before opening a pull request, make sure all of these pass locally — CI runs
the same checks and will block otherwise:

- `gofmt -l .` prints nothing (code is formatted)
- `go vet ./...` is clean
- `go test -race ./...` passes
- `go mod tidy` produces no diff

Run `gofmt -w .` to format.

## Project layout

```
cmd/canvux        CLI entrypoint (editor + subcommands)
internal/app      the interactive editor: tools, input, overlays, UI
internal/scene    scene graph: objects, layers, hit-testing, serialization
internal/render   rasterizer + terminal compositors + color profiles
internal/svg      SVG export/import
internal/export   PNG export
internal/imgembed image → vector rects
internal/config   layered JSON configuration
internal/plugin   external-process plugin protocol
internal/collab   realtime collaboration server + client
internal/clipboard system clipboard (native + OSC 52)
internal/history  undo/redo
internal/geom     2D vector math
```

The rasterizer draws into an abstract `render.Surface`, so the terminal buffer
and the PNG exporter share the same drawing code — new shape kinds only need to
be implemented once.

## Conventions

- **Tests** accompany behavior changes. Pure logic (geometry, scene graph,
  export, config) has unit tests; UI flows are covered by PTY-driven smoke
  tests where practical.
- **Keep commits focused** and write a clear message explaining the *why*.
- **Match the surrounding style** — comment density, naming, and idiom.
- New user-facing actions go through the shared `actionTable()` in
  `internal/app/actions.go` so they appear in the command palette and are
  rebindable; don't add bare `switch` cases in the key handler.
- Shipped example files (e.g. `examples/config.example.json`) are covered by
  drift tests — update them alongside the code they document.

## Pull requests

1. Branch from `main` with a descriptive name (e.g. `feat/snap-to-guides`, not
   `patch-1`).
2. Push and open a PR against `main`. CI (build/test + audit) must be green.
3. `@anishhs-gh` is the code owner and reviews all PRs.

## Reporting issues

Please include the Canvux version (`canvux version`), your OS and terminal, and
steps to reproduce. For rendering problems, note your `TERM`/`COLORTERM` and
whether `--color` changes the behavior.

## License

By contributing, you agree that your contributions are licensed under the
project's [MIT License](LICENSE).
