// Command canvux is a terminal-native infinite vector canvas.
//
//	canvux [file.canvux]              open the editor
//	canvux export <file> [flags]      export to SVG or PNG
//	canvux info <file>                print document stats
//	canvux version
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/anishhs-gh/canvux/internal/app"
	"github.com/anishhs-gh/canvux/internal/render"
	"github.com/anishhs-gh/canvux/internal/scene"
	"github.com/anishhs-gh/canvux/internal/svg"
)

// version is stamped at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	args := os.Args[1:]
	if len(args) > 0 {
		switch args[0] {
		case "export":
			exitOn(runExport(args[1:]))
			return
		case "render":
			exitOn(runRender(args[1:]))
			return
		case "add":
			exitOn(runAdd(args[1:]))
			return
		case "info":
			exitOn(runInfo(args[1:]))
			return
		case "diff":
			differs, err := runDiff(args[1:])
			exitOn(err)
			if differs {
				os.Exit(1)
			}
			return
		case "serve":
			exitOn(runServe(args[1:]))
			return
		case "join":
			exitOn(runJoin(args[1:]))
			return
		case "plugins":
			exitOn(runPlugins())
			return
		case "version", "--version", "-v":
			fmt.Printf("canvux %s\n", version)
			return
		case "help", "--help", "-h":
			usage()
			return
		}
	}
	path := ""
	if len(args) > 0 {
		path = args[0]
		if strings.HasPrefix(path, "-") {
			usage()
			os.Exit(2)
		}
	}
	m, err := app.New(path)
	exitOn(err)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseAllMotion())
	_, err = p.Run()
	exitOn(err)
}

func runExport(args []string) error {
	fs := flag.NewFlagSet("export", flag.ExitOnError)
	outSVG := fs.String("svg", "", "output SVG path (default: <input>.svg)")
	outPNG := fs.String("png", "", "output PNG path")
	scale := fs.Float64("scale", 8, "PNG pixels per world unit")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: canvux export <file.canvux> [--svg out.svg] [--png out.png] [--scale N]")
		fs.PrintDefaults()
	}
	// Accept the input file before or after flags (every flag takes a value).
	var pos, flagArgs []string
	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "-") {
			flagArgs = append(flagArgs, args[i])
			if !strings.Contains(args[i], "=") && i+1 < len(args) {
				flagArgs = append(flagArgs, args[i+1])
				i++
			}
		} else {
			pos = append(pos, args[i])
		}
	}
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}
	if len(pos) != 1 {
		fs.Usage()
		return fmt.Errorf("expected exactly one input file")
	}
	in := pos[0]
	doc, err := scene.Load(in)
	if err != nil {
		return err
	}
	if *outSVG == "" && *outPNG == "" {
		*outSVG = strings.TrimSuffix(in, filepath.Ext(in)) + ".svg"
	}
	if *outSVG != "" {
		if err := os.WriteFile(*outSVG, svg.Export(doc), 0o644); err != nil {
			return err
		}
		fmt.Printf("wrote %s\n", *outSVG)
	}
	if *outPNG != "" {
		if err := app.ExportPNG(doc, *outPNG, *scale); err != nil {
			return err
		}
		fmt.Printf("wrote %s\n", *outPNG)
	}
	return nil
}

func runRender(args []string) error {
	fs := flag.NewFlagSet("render", flag.ExitOnError)
	cols := fs.Int("cols", 100, "output width in terminal columns")
	rows := fs.Int("rows", 30, "output height in terminal rows")
	braille := fs.Bool("braille", false, "use hi-res braille rendering")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: canvux render <file.canvux> [--cols N] [--rows N] [--braille]")
		fs.PrintDefaults()
	}
	var pos, flagArgs []string
	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "-") {
			flagArgs = append(flagArgs, args[i])
			if !strings.Contains(args[i], "=") && i+1 < len(args) && !strings.HasPrefix(args[i], "--braille") {
				flagArgs = append(flagArgs, args[i+1])
				i++
			}
		} else {
			pos = append(pos, args[i])
		}
	}
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}
	if len(pos) != 1 {
		fs.Usage()
		return fmt.Errorf("expected exactly one input file")
	}
	doc, err := scene.Load(pos[0])
	if err != nil {
		return err
	}
	mode := render.ModeHalfBlock
	if *braille {
		mode = render.ModeBraille
	}
	fmt.Println(app.RenderFrame(doc, *cols, *rows, mode))
	return nil
}

func runInfo(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: canvux info <file.canvux>")
	}
	doc, err := scene.Load(args[0])
	if err != nil {
		return err
	}
	kinds := map[scene.Kind]int{}
	for _, o := range doc.Objects {
		kinds[o.Kind]++
	}
	b := doc.ContentBounds()
	fmt.Printf("file:     %s\n", args[0])
	fmt.Printf("version:  %d\n", doc.Version)
	fmt.Printf("layers:   %d\n", len(doc.Layers))
	fmt.Printf("objects:  %d\n", len(doc.Objects))
	for k, n := range kinds {
		fmt.Printf("  %-9s %d\n", k, n)
	}
	fmt.Printf("bounds:   (%.1f, %.1f) – (%.1f, %.1f)  %.1f × %.1f\n",
		b.Min.X, b.Min.Y, b.Max.X, b.Max.Y, b.W(), b.H())
	return nil
}

func usage() {
	fmt.Print(`canvux — terminal-native infinite vector canvas

usage:
  canvux [file.canvux]     open the editor (creates the file on save)
  canvux export <file>     export to SVG and/or PNG
      --svg out.svg        SVG output path (default <input>.svg)
      --png out.png        PNG output path
      --scale N            PNG pixels per world unit (default 8)
  canvux render <file>     print one frame to stdout (like cat for drawings)
      --cols N --rows N    output size (default 100x30)
      --braille            hi-res braille rendering
  canvux add <kind> <file> scripted insertion: rect|ellipse|line|arrow|text|
                           polygon|image (see canvux add --help)
  canvux info <file>       print document statistics
  canvux diff <a> <b>      object-level diff; exit 1 when files differ
  canvux serve <file>      host a realtime collaboration session
  canvux join <host:port>  open the editor connected to a session
  canvux plugins           list discovered canvux-* plugins
  canvux version

editor quick keys: ? help · : command palette · ctrl+s save · q quit
`)
}

func exitOn(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "canvux:", err)
		os.Exit(1)
	}
}
