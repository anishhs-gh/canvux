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
	case "path":
		pts, closed := flattenPathD(get("d"))
		if len(pts) < 2 {
			return nil, false
		}
		if closed {
			o.Kind = scene.KindPolygon
		} else {
			o.Kind = scene.KindPath
		}
		o.Points = pts
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

// flattenPathD converts an SVG path `d` attribute into a polyline, flattening
// curves. Supports M/m, L/l, H/h, V/v, C/c, Q/q, Z/z; anything else aborts.
// Returns the points and whether the path was closed.
func flattenPathD(d string) ([]geom.Vec, bool) {
	toks := tokenizePath(d)
	var pts []geom.Vec
	var cur, start geom.Vec
	closed := false
	i := 0
	next := func() (float64, bool) {
		if i < len(toks) {
			if v, err := strconv.ParseFloat(toks[i], 64); err == nil {
				i++
				return v, true
			}
		}
		return 0, false
	}
	pair := func() (geom.Vec, bool) {
		x, ok1 := next()
		y, ok2 := next()
		return geom.V(x, y), ok1 && ok2
	}
	cmd := ""
	for i < len(toks) {
		t := toks[i]
		isLetter := len(t) == 1 && (t[0] >= 'A' && t[0] <= 'Z' || t[0] >= 'a' && t[0] <= 'z')
		if isLetter {
			if !strings.ContainsAny(t, "MmLlHhVvCcQqZz") {
				return nil, false // arcs, S/T shorthands: unsupported
			}
			cmd = t
			i++
		} else if cmd == "" {
			return nil, false
		}
		rel := cmd >= "a" // lowercase = relative
		switch strings.ToUpper(cmd) {
		case "M", "L":
			p, ok := pair()
			if !ok {
				return pts, closed
			}
			if rel {
				p = p.Add(cur)
			}
			cur = p
			if strings.ToUpper(cmd) == "M" && len(pts) == 0 {
				start = p
			}
			pts = append(pts, p)
			if strings.ToUpper(cmd) == "M" {
				cmd = strings.Replace(cmd, "M", "L", 1)
				cmd = strings.Replace(cmd, "m", "l", 1)
			}
		case "H", "V":
			v, ok := next()
			if !ok {
				return pts, closed
			}
			p := cur
			if strings.ToUpper(cmd) == "H" {
				if rel {
					p.X += v
				} else {
					p.X = v
				}
			} else {
				if rel {
					p.Y += v
				} else {
					p.Y = v
				}
			}
			cur = p
			pts = append(pts, p)
		case "C":
			c1, ok1 := pair()
			c2, ok2 := pair()
			end, ok3 := pair()
			if !ok1 || !ok2 || !ok3 {
				return pts, closed
			}
			if rel {
				c1, c2, end = c1.Add(cur), c2.Add(cur), end.Add(cur)
			}
			pts = append(pts, flattenCubic(cur, c1, c2, end, 16)...)
			cur = end
		case "Q":
			c1, ok1 := pair()
			end, ok2 := pair()
			if !ok1 || !ok2 {
				return pts, closed
			}
			if rel {
				c1, end = c1.Add(cur), end.Add(cur)
			}
			// Elevate quadratic to cubic.
			cc1 := cur.Add(c1.Sub(cur).Mul(2.0 / 3.0))
			cc2 := end.Add(c1.Sub(end).Mul(2.0 / 3.0))
			pts = append(pts, flattenCubic(cur, cc1, cc2, end, 12)...)
			cur = end
		case "Z":
			closed = true
			cur = start
		default:
			return nil, false // arcs, S/T shorthands: unsupported
		}
	}
	return pts, closed
}

func flattenCubic(p0, c1, c2, p1 geom.Vec, n int) []geom.Vec {
	out := make([]geom.Vec, n)
	for k := 1; k <= n; k++ {
		t := float64(k) / float64(n)
		u := 1 - t
		out[k-1] = geom.V(
			u*u*u*p0.X+3*u*u*t*c1.X+3*u*t*t*c2.X+t*t*t*p1.X,
			u*u*u*p0.Y+3*u*u*t*c1.Y+3*u*t*t*c2.Y+t*t*t*p1.Y,
		)
	}
	return out
}

// tokenizePath splits a path data string into command letters and numbers.
func tokenizePath(d string) []string {
	var toks []string
	var num strings.Builder
	flush := func() {
		if num.Len() > 0 {
			toks = append(toks, num.String())
			num.Reset()
		}
	}
	for _, r := range d {
		switch {
		case r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z':
			flush()
			toks = append(toks, string(r))
		case r == ' ' || r == ',' || r == '\n' || r == '\t' || r == '\r':
			flush()
		case r == '-' && num.Len() > 0 && !strings.HasSuffix(num.String(), "e"):
			flush()
			num.WriteRune(r)
		default:
			num.WriteRune(r)
		}
	}
	flush()
	return toks
}
