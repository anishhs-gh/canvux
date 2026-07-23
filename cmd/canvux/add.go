package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/anishhs-gh/canvux/internal/geom"
	"github.com/anishhs-gh/canvux/internal/imgembed"
	"github.com/anishhs-gh/canvux/internal/scene"
)

// runAdd implements `canvux add <kind> <file.canvux> [flags]` — scripted
// object insertion (roadmap phase 12).
func runAdd(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: canvux add <rect|ellipse|line|arrow|text|polygon|image> <file.canvux> [flags]")
	}
	kind := args[0]
	fs := flag.NewFlagSet("add", flag.ExitOnError)
	at := fs.String("at", "0,0", "position x,y (top-left / line start)")
	to := fs.String("to", "", "line/arrow end x,y")
	size := fs.String("size", "20,10", "width,height for rect/ellipse")
	text := fs.String("text", "", "text content (kind text)")
	points := fs.String("points", "", "polygon points: \"x,y x,y x,y\"")
	stroke := fs.String("stroke", "#c0caf5", "stroke color #rrggbb")
	fill := fs.String("fill", "", "fill color #rrggbb (enables fill)")
	width := fs.Float64("width", 1, "stroke width")
	layer := fs.Int("layer", 0, "target layer index")
	src := fs.String("src", "", "image path (kind image)")
	cols := fs.Int("cols", 48, "max image width in world cells (kind image)")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: canvux add <kind> <file.canvux> [flags]")
		fs.PrintDefaults()
	}
	var pos, flagArgs []string
	for i := 1; i < len(args); i++ {
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
		return fmt.Errorf("expected exactly one .canvux file")
	}
	path := pos[0]

	doc, err := scene.Load(path)
	if os.IsNotExist(err) {
		doc, err = scene.NewDoc(), nil
	}
	if err != nil {
		return err
	}
	origin, err := parseXY(*at)
	if err != nil {
		return fmt.Errorf("--at: %w", err)
	}

	var strokeC, fillC scene.Color
	if err := strokeC.UnmarshalText([]byte(*stroke)); err != nil {
		return fmt.Errorf("--stroke: %w", err)
	}
	filled := *fill != ""
	if filled {
		if err := fillC.UnmarshalText([]byte(*fill)); err != nil {
			return fmt.Errorf("--fill: %w", err)
		}
	}

	base := &scene.Object{
		Stroke: strokeC, Fill: fillC, Filled: filled,
		StrokeWidth: *width, Opacity: 1, Layer: *layer,
	}
	var added int
	switch kind {
	case "rect", "ellipse":
		sz, err := parseXY(*size)
		if err != nil {
			return fmt.Errorf("--size: %w", err)
		}
		base.Kind = scene.KindRect
		if kind == "ellipse" {
			base.Kind = scene.KindEllipse
		}
		base.P1, base.P2 = origin, origin.Add(sz)
		doc.Add(base)
		added = 1
	case "line", "arrow":
		if *to == "" {
			return fmt.Errorf("--to is required for %s", kind)
		}
		end, err := parseXY(*to)
		if err != nil {
			return fmt.Errorf("--to: %w", err)
		}
		base.Kind = scene.KindLine
		if kind == "arrow" {
			base.Kind = scene.KindArrow
		}
		base.P1, base.P2 = origin, end
		doc.Add(base)
		added = 1
	case "text":
		if *text == "" {
			return fmt.Errorf("--text is required")
		}
		base.Kind = scene.KindText
		base.P1 = origin
		base.Text = *text
		doc.Add(base)
		added = 1
	case "polygon":
		for _, part := range strings.Fields(*points) {
			p, err := parseXY(part)
			if err != nil {
				return fmt.Errorf("--points: %w", err)
			}
			base.Points = append(base.Points, origin.Add(p))
		}
		if len(base.Points) < 3 {
			return fmt.Errorf("--points needs at least 3 points")
		}
		base.Kind = scene.KindPolygon
		doc.Add(base)
		added = 1
	case "image":
		if *src == "" {
			return fmt.Errorf("--src is required for image")
		}
		objs, err := imgembed.FromFile(*src, *cols, *layer)
		if err != nil {
			return err
		}
		for _, o := range objs {
			o.Translate(origin)
			doc.Add(o)
		}
		added = len(objs)
	default:
		return fmt.Errorf("unknown kind %q", kind)
	}
	if err := doc.Save(path); err != nil {
		return err
	}
	fmt.Printf("added %d object(s) to %s (%d total)\n", added, path, len(doc.Objects))
	return nil
}

func parseXY(s string) (geom.Vec, error) {
	x, y, ok := strings.Cut(strings.TrimSpace(s), ",")
	if !ok {
		return geom.Vec{}, fmt.Errorf("want x,y — got %q", s)
	}
	fx, err1 := strconv.ParseFloat(strings.TrimSpace(x), 64)
	fy, err2 := strconv.ParseFloat(strings.TrimSpace(y), 64)
	if err1 != nil || err2 != nil {
		return geom.Vec{}, fmt.Errorf("want numeric x,y — got %q", s)
	}
	return geom.V(fx, fy), nil
}
