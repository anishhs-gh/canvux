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
	for _, o := range d.VisibleObjects() {
		writeObject(&b, o)
	}
	b.WriteString("</svg>\n")
	return []byte(b.String())
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
		fmt.Fprintf(b, `  <polyline points="%s"%s%s/>`+"\n", pointList(o.Points), style, tr)
	case scene.KindText:
		fmt.Fprintf(b, `  <text x="%s" y="%s" font-family="monospace" font-size="2" fill="%s"%s%s>%s</text>`+"\n",
			f(o.P1.X), f(o.P1.Y+1.6), o.Stroke.Hex(), opacityAttr(o), tr, escape(o.Text))
	}
}

func styleAttrs(o *scene.Object) string {
	var b strings.Builder
	fill := "none"
	if o.Filled {
		fill = o.Fill.Hex()
	}
	fmt.Fprintf(&b, ` fill="%s" stroke="%s" stroke-width="%s"`, fill, o.Stroke.Hex(), f(o.StrokeWidth))
	if o.Dashed {
		b.WriteString(` stroke-dasharray="2 2"`)
	}
	b.WriteString(opacityAttr(o))
	b.WriteString(` stroke-linecap="round" stroke-linejoin="round"`)
	return b.String()
}

func opacityAttr(o *scene.Object) string {
	if o.Opacity < 1 {
		return fmt.Sprintf(` opacity="%s"`, f(o.Opacity))
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
