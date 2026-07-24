// Package svg exports Canvux documents to standards-compliant SVG and
// imports basic SVG shapes back into the scene graph.
package svg

import (
	"fmt"
	"math"
	"strings"

	"github.com/anishhs-gh/canvux/internal/geom"
	"github.com/anishhs-gh/canvux/internal/scene"
)

// Export renders the document's visible objects as an SVG file.
func Export(d *scene.Doc) []byte {
	var body, defs strings.Builder
	for _, o := range d.VisibleObjects() {
		writeDefs(&defs, o)
		writeObject(&body, o)
	}

	var b strings.Builder
	bounds := d.ContentBounds().Expand(4)
	if bounds.W() <= 0 || bounds.H() <= 0 {
		bounds = geom.R(geom.V(0, 0), geom.V(100, 100))
	}
	fmt.Fprintf(&b, `<?xml version="1.0" encoding="UTF-8"?>`+"\n")
	fmt.Fprintf(&b,
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="%s %s %s %s" width="%s" height="%s">`+"\n",
		f(bounds.Min.X), f(bounds.Min.Y), f(bounds.W()), f(bounds.H()), f(bounds.W()*8), f(bounds.H()*8))
	b.WriteString("  <desc>Created with Canvux</desc>\n")
	if defs.Len() > 0 {
		b.WriteString("  <defs>\n" + defs.String() + "  </defs>\n")
	}
	b.WriteString(body.String())
	b.WriteString("</svg>\n")
	return []byte(b.String())
}

// writeDefs emits the object's gradient and filter definitions, if any.
func writeDefs(b *strings.Builder, o *scene.Object) {
	if o.Fill2 != nil && o.Filled {
		ang := o.GradAngle * math.Pi / 180
		if o.GradAngle == 0 {
			ang = math.Pi / 2
		}
		ux, uy := math.Cos(ang), math.Sin(ang)
		fmt.Fprintf(b, `    <linearGradient id="grad%d" x1="%s" y1="%s" x2="%s" y2="%s">`+"\n",
			o.ID, f(0.5-ux/2), f(0.5-uy/2), f(0.5+ux/2), f(0.5+uy/2))
		fmt.Fprintf(b, `      <stop offset="0" stop-color="%s"/>`+"\n", o.Fill.Hex())
		fmt.Fprintf(b, `      <stop offset="1" stop-color="%s"/>`+"\n", o.Fill2.Hex())
		b.WriteString("    </linearGradient>\n")
	}
	if o.Shadow || o.Blur > 0 {
		fmt.Fprintf(b, `    <filter id="fx%d" x="-40%%" y="-40%%" width="180%%" height="180%%">`+"\n", o.ID)
		if o.Shadow {
			fmt.Fprintf(b, `      <feDropShadow dx="%s" dy="%s" stdDeviation="0.8" flood-color="#050609" flood-opacity="0.45"/>`+"\n",
				f(1.2), f(1.2))
		}
		if o.Blur > 0 {
			fmt.Fprintf(b, `      <feGaussianBlur stdDeviation="%s"/>`+"\n", f(o.Blur/2))
		}
		b.WriteString("    </filter>\n")
	}
}

func writeObject(b *strings.Builder, o *scene.Object) {
	style := styleAttrs(o)
	tr := ""
	if o.Rotation != 0 {
		c := o.Center()
		tr = fmt.Sprintf(` transform="rotate(%s %s %s)"`, f(o.Rotation*180/math.Pi), f(c.X), f(c.Y))
	}
	switch o.Kind {
	case scene.KindLine:
		fmt.Fprintf(b, `  <line x1="%s" y1="%s" x2="%s" y2="%s"%s%s/>`+"\n",
			f(o.P1.X), f(o.P1.Y), f(o.P2.X), f(o.P2.Y), style, tr)
	case scene.KindBezier:
		fmt.Fprintf(b, `  <path d="M %s %s C %s %s, %s %s, %s %s"%s%s/>`+"\n",
			f(o.P1.X), f(o.P1.Y), f(o.C1.X), f(o.C1.Y), f(o.C2.X), f(o.C2.Y), f(o.P2.X), f(o.P2.Y), style, tr)
	case scene.KindArrow:
		fmt.Fprintf(b, `  <g%s%s>`+"\n", style, tr)
		fmt.Fprintf(b, `    <line x1="%s" y1="%s" x2="%s" y2="%s"/>`+"\n",
			f(o.P1.X), f(o.P1.Y), f(o.P2.X), f(o.P2.Y))
		if head := arrowHead(o); head != "" {
			fmt.Fprintf(b, `    <polygon points="%s" fill="%s" stroke="none"/>`+"\n", head, o.Stroke.Hex())
		}
		b.WriteString("  </g>\n")
	case scene.KindRect:
		r := geom.R(o.P1, o.P2)
		fmt.Fprintf(b, `  <rect x="%s" y="%s" width="%s" height="%s"%s%s/>`+"\n",
			f(r.Min.X), f(r.Min.Y), f(r.W()), f(r.H()), style, tr)
	case scene.KindEllipse:
		r := geom.R(o.P1, o.P2)
		c := r.Center()
		fmt.Fprintf(b, `  <ellipse cx="%s" cy="%s" rx="%s" ry="%s"%s%s/>`+"\n",
			f(c.X), f(c.Y), f(r.W()/2), f(r.H()/2), style, tr)
	case scene.KindPolygon:
		fmt.Fprintf(b, `  <polygon points="%s"%s%s/>`+"\n", pointList(o.Points), style, tr)
	case scene.KindPath:
		if len(o.Widths) == len(o.Points) && len(o.Points) > 1 {
			writeVariableWidthPath(b, o, tr)
		} else {
			fmt.Fprintf(b, `  <polyline points="%s"%s%s/>`+"\n", pointList(o.Points), style, tr)
		}
	case scene.KindText:
		lines := scene.TextLines(o.Text)
		if len(lines) == 1 {
			fmt.Fprintf(b, `  <text x="%s" y="%s" font-family="monospace" font-size="2" fill="%s"%s%s%s>%s</text>`+"\n",
				f(o.P1.X), f(o.P1.Y+1.6), o.Stroke.Hex(), opacityAttr(o), filterAttr(o), tr, escape(o.Text))
		} else {
			// Multi-line: one <tspan> per line, stepped down by the line height.
			fmt.Fprintf(b, `  <text x="%s" y="%s" font-family="monospace" font-size="2" fill="%s"%s%s%s>`+"\n",
				f(o.P1.X), f(o.P1.Y+1.6), o.Stroke.Hex(), opacityAttr(o), filterAttr(o), tr)
			for i, ln := range lines {
				dy := "0"
				if i > 0 {
					dy = "2"
				}
				fmt.Fprintf(b, `    <tspan x="%s" dy="%s">%s</tspan>`+"\n", f(o.P1.X), dy, escape(ln))
			}
			b.WriteString("  </text>\n")
		}
	}
}

// writeVariableWidthPath splits a variable-width brush stroke into runs of
// similar width, one polyline each, so the taper survives export.
func writeVariableWidthPath(b *strings.Builder, o *scene.Object, tr string) {
	fmt.Fprintf(b, `  <g stroke="%s" fill="none" stroke-linecap="round" stroke-linejoin="round"%s%s%s>`+"\n",
		o.Stroke.Hex(), opacityAttr(o), filterAttr(o), tr)
	quant := func(w float64) float64 { return math.Round(w*4) / 4 }
	start := 0
	for i := 1; i < len(o.Points); i++ {
		if i == len(o.Points)-1 || quant(o.Widths[i]) != quant(o.Widths[start]) {
			run := o.Points[start : i+1]
			fmt.Fprintf(b, `    <polyline points="%s" stroke-width="%s"/>`+"\n",
				pointList(run), f(math.Max(0.2, quant(o.Widths[start]))))
			start = i
		}
	}
	b.WriteString("  </g>\n")
}

func styleAttrs(o *scene.Object) string {
	var b strings.Builder
	fill := "none"
	if o.Filled {
		if o.Fill2 != nil {
			fill = fmt.Sprintf("url(#grad%d)", o.ID)
		} else {
			fill = o.Fill.Hex()
		}
	}
	fmt.Fprintf(&b, ` fill="%s" stroke="%s" stroke-width="%s"`, fill, o.Stroke.Hex(), f(o.StrokeWidth))
	if o.Dashed {
		b.WriteString(` stroke-dasharray="2 2"`)
	}
	b.WriteString(opacityAttr(o))
	b.WriteString(filterAttr(o))
	b.WriteString(` stroke-linecap="round" stroke-linejoin="round"`)
	return b.String()
}

func opacityAttr(o *scene.Object) string {
	if o.Opacity < 1 {
		return fmt.Sprintf(` opacity="%s"`, f(o.Opacity))
	}
	return ""
}

func filterAttr(o *scene.Object) string {
	if o.Shadow || o.Blur > 0 {
		return fmt.Sprintf(` filter="url(#fx%d)"`, o.ID)
	}
	return ""
}

func arrowHead(o *scene.Object) string {
	dir := o.P2.Sub(o.P1)
	if dir.Len() == 0 {
		return ""
	}
	n := dir.Mul(1 / dir.Len())
	size := math.Max(1.5, math.Min(3, dir.Len()*0.3))
	base := o.P2.Sub(n.Mul(size))
	perp := geom.V(-n.Y, n.X).Mul(size * 0.5)
	return pointList([]geom.Vec{o.P2, base.Add(perp), base.Sub(perp)})
}

func pointList(pts []geom.Vec) string {
	parts := make([]string, len(pts))
	for i, p := range pts {
		parts[i] = f(p.X) + "," + f(p.Y)
	}
	return strings.Join(parts, " ")
}

func f(v float64) string {
	s := fmt.Sprintf("%.3f", v)
	s = strings.TrimRight(s, "0")
	return strings.TrimRight(s, ".")
}

func escape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return r.Replace(s)
}
