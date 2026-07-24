package geom

// Simplify reduces a polyline using the Ramer–Douglas–Peucker algorithm,
// dropping points that lie within eps of the line between kept points. The
// endpoints are always preserved. eps is in world units.
func Simplify(pts []Vec, eps float64) []Vec {
	if len(pts) < 3 || eps <= 0 {
		return pts
	}
	keep := make([]bool, len(pts))
	keep[0], keep[len(pts)-1] = true, true
	rdp(pts, 0, len(pts)-1, eps, keep)
	out := pts[:0:0]
	for i, k := range keep {
		if k {
			out = append(out, pts[i])
		}
	}
	return out
}

func rdp(pts []Vec, lo, hi int, eps float64, keep []bool) {
	if hi <= lo+1 {
		return
	}
	var maxD float64
	maxI := lo
	for i := lo + 1; i < hi; i++ {
		if d := DistToSegment(pts[i], pts[lo], pts[hi]); d > maxD {
			maxD, maxI = d, i
		}
	}
	if maxD > eps {
		keep[maxI] = true
		rdp(pts, lo, maxI, eps, keep)
		rdp(pts, maxI, hi, eps, keep)
	}
}

// Chaikin smooths a polyline by iterations of corner-cutting. Each iteration
// replaces every interior segment with two points at 1/4 and 3/4, rounding off
// corners. Endpoints are preserved. Good for freehand strokes.
func Chaikin(pts []Vec, iterations int) []Vec {
	for it := 0; it < iterations && len(pts) >= 3; it++ {
		out := make([]Vec, 0, len(pts)*2)
		out = append(out, pts[0])
		for i := 0; i+1 < len(pts); i++ {
			p, q := pts[i], pts[i+1]
			out = append(out, p.Lerp(q, 0.25), p.Lerp(q, 0.75))
		}
		out = append(out, pts[len(pts)-1])
		pts = out
	}
	return pts
}

// SmoothStroke cleans up a freehand stroke: decimate to drop jitter, then
// smooth the corners. tol is the decimation tolerance in world units.
func SmoothStroke(pts []Vec, tol float64) []Vec {
	if len(pts) < 3 {
		return pts
	}
	pts = Simplify(pts, tol)
	// One or two Chaikin passes are plenty; more just inflates the point count.
	if len(pts) >= 3 {
		pts = Chaikin(pts, 2)
	}
	return pts
}

// PolylineLength returns the total length of a polyline.
func PolylineLength(pts []Vec) float64 {
	var sum float64
	for i := 0; i+1 < len(pts); i++ {
		sum += pts[i].Dist(pts[i+1])
	}
	return sum
}
