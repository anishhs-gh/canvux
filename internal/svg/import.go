package svg

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"

	"github.com/anishhs-gh/canvux/internal/geom"
	"github.com/anishhs-gh/canvux/internal/scene"
)

// Import parses basic SVG shapes (rect, circle, ellipse, line, polyline,
// polygon, text) into a new document. Groups are flattened; unsupported
// elements are skipped and counted.
func Import(data []byte) (*scene.Doc, int, error) {
	doc := scene.NewDoc()
	dec := xml.NewDecoder(strings.NewReader(string(data)))
	skipped := 0
	var textEl *scene.Object // pending <text> awaiting char data
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			obj, ok := elementToObject(t)
			if obj != nil {
				if obj.Kind == scene.KindText {
					textEl = obj
				} else {
					doc.Add(obj)
				}
			} else if !ok {
				skipped++
			}
		case xml.CharData:
			if textEl != nil {
				textEl.Text += strings.TrimSpace(string(t))
			}
		case xml.EndElement:
			if t.Name.Local == "text" && textEl != nil {
				if textEl.Text != "" {
					doc.Add(textEl)
				}
				textEl = nil
			}
		}
	}
	if len(doc.Objects) == 0 && skipped == 0 {
		return nil, 0, fmt.Errorf("no supported SVG elements found")
	}
	return doc, skipped, nil
}

// elementToObject converts one SVG element. Returns (nil, true) for
// structural elements that are fine to ignore, (nil, false) for unsupported
// drawing elements.
func elementToObject(el xml.StartElement) (*scene.Object, bool) {
	get := func(name string) string {
		for _, a := range el.Attr {
			if a.Name.Local == name {
				return a.Value
			}
		}
		return ""
	}
	num := func(name string) float64 {
		v, _ := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSpace(get(name)), "px"), 64)
		return v
	}
	o := &scene.Object{Stroke: scene.Color{R: 220, G: 220, B: 220}, StrokeWidth: 1, Opacity: 1}
	applyStyle(o, el)

	switch el.Name.Local {
	case "svg", "g", "desc", "title", "defs", "metadata":
		return nil, true
	case "rect":
		o.Kind = scene.KindRect
		o.P1 = geom.V(num("x"), num("y"))
		o.P2 = o.P1.Add(geom.V(num("width"), num("height")))
	case "circle":
		o.Kind = scene.KindEllipse
		c, r := geom.V(num("cx"), num("cy")), num("r")
		o.P1, o.P2 = c.Sub(geom.V(r, r)), c.Add(geom.V(r, r))
	case "ellipse":
		o.Kind = scene.KindEllipse
		c := geom.V(num("cx"), num("cy"))
		r := geom.V(num("rx"), num("ry"))
		o.P1, o.P2 = c.Sub(r), c.Add(r)
	case "line":
		o.Kind = scene.KindLine
		o.P1 = geom.V(num("x1"), num("y1"))
		o.P2 = geom.V(num("x2"), num("y2"))
	case "polyline":
		o.Kind = scene.KindPath
		o.Points = parsePoints(get("points"))
	case "polygon":
		o.Kind = scene.KindPolygon
		o.Points = parsePoints(get("points"))
	case "text":
		o.Kind = scene.KindText
		o.P1 = geom.V(num("x"), num("y")-1.6)
	default:
		return nil, false
	}
	if (o.Kind == scene.KindPath || o.Kind == scene.KindPolygon) && len(o.Points) < 2 {
		return nil, false
	}
	return o, true
}

func applyStyle(o *scene.Object, el xml.StartElement) {
	attrs := map[string]string{}
	for _, a := range el.Attr {
		attrs[a.Name.Local] = a.Value
	}
	// Inline style="" wins over presentation attributes.
	if style, ok := attrs["style"]; ok {
		for _, decl := range strings.Split(style, ";") {
			if k, v, ok := strings.Cut(decl, ":"); ok {
				attrs[strings.TrimSpace(k)] = strings.TrimSpace(v)
			}
		}
	}
	if v, ok := attrs["stroke"]; ok {
		if c, err := parseColor(v); err == nil {
			o.Stroke = c
		}
	}
	if v, ok := attrs["fill"]; ok && v != "none" {
		if c, err := parseColor(v); err == nil {
			o.Fill = c
			o.Filled = true
		}
	}
	if v, ok := attrs["stroke-width"]; ok {
		if w, err := strconv.ParseFloat(strings.TrimSuffix(v, "px"), 64); err == nil && w > 0 {
			o.StrokeWidth = w
		}
	}
	if v, ok := attrs["opacity"]; ok {
		if a, err := strconv.ParseFloat(v, 64); err == nil && a > 0 && a <= 1 {
			o.Opacity = a
		}
	}
	if v, ok := attrs["stroke-dasharray"]; ok && v != "none" && v != "" {
		o.Dashed = true
	}
	// Filled shapes with no explicit stroke: use the fill color for the outline.
	if o.Filled {
		if _, ok := attrs["stroke"]; !ok {
			o.Stroke = o.Fill
		}
	}
}

func parseColor(s string) (scene.Color, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if named, ok := cssColors[s]; ok {
		s = named
	}
	if strings.HasPrefix(s, "#") && len(s) == 4 {
		s = fmt.Sprintf("#%c%c%c%c%c%c", s[1], s[1], s[2], s[2], s[3], s[3])
	}
	if strings.HasPrefix(s, "rgb(") && strings.HasSuffix(s, ")") {
		parts := strings.Split(s[4:len(s)-1], ",")
		if len(parts) == 3 {
			var v [3]int
			for i, p := range parts {
				v[i], _ = strconv.Atoi(strings.TrimSpace(p))
			}
			return scene.Color{R: uint8(v[0]), G: uint8(v[1]), B: uint8(v[2])}, nil
		}
	}
	var c scene.Color
	err := c.UnmarshalText([]byte(s))
	return c, err
}

var cssColors = map[string]string{
	"black": "#000000", "white": "#ffffff", "red": "#ff0000", "green": "#008000",
	"blue": "#0000ff", "yellow": "#ffff00", "cyan": "#00ffff", "magenta": "#ff00ff",
	"gray": "#808080", "grey": "#808080", "orange": "#ffa500", "purple": "#800080",
	"pink": "#ffc0cb", "lime": "#00ff00", "navy": "#000080", "teal": "#008080",
	"silver": "#c0c0c0", "maroon": "#800000", "olive": "#808000",
}

func parsePoints(s string) []geom.Vec {
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return r == ' ' || r == ',' || r == '\n' || r == '\t' || r == '\r'
	})
	var pts []geom.Vec
	for i := 0; i+1 < len(fields); i += 2 {
		x, err1 := strconv.ParseFloat(fields[i], 64)
		y, err2 := strconv.ParseFloat(fields[i+1], 64)
		if err1 == nil && err2 == nil {
			pts = append(pts, geom.V(x, y))
		}
	}
	return pts
}
