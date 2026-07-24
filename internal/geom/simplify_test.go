package geom

import (
	"math"
	"testing"
)

func TestSimplifyCollinear(t *testing.T) {
	// A straight run of points collapses to its two endpoints.
	pts := []Vec{{0, 0}, {1, 0}, {2, 0}, {3, 0}, {4, 0}}
	got := Simplify(pts, 0.01)
	if len(got) != 2 || got[0] != (Vec{0, 0}) || got[1] != (Vec{4, 0}) {
		t.Fatalf("collinear simplify = %v, want two endpoints", got)
	}
}

func TestSimplifyKeepsCorners(t *testing.T) {
	// An L-shape must keep the corner.
	pts := []Vec{{0, 0}, {1, 0}, {2, 0}, {2, 1}, {2, 2}}
	got := Simplify(pts, 0.1)
	if len(got) != 3 {
		t.Fatalf("L-shape simplify = %v (%d pts), want 3", got, len(got))
	}
	if got[1] != (Vec{2, 0}) {
		t.Errorf("corner not preserved: %v", got)
	}
}

func TestSimplifyShortInputUnchanged(t *testing.T) {
	pts := []Vec{{0, 0}, {5, 5}}
	if got := Simplify(pts, 1); len(got) != 2 {
		t.Errorf("2-point input changed: %v", got)
	}
}

func TestChaikinRoundsAndGrows(t *testing.T) {
	pts := []Vec{{0, 0}, {2, 0}, {2, 2}}
	got := Chaikin(pts, 1)
	if len(got) <= len(pts) {
		t.Errorf("Chaikin should add points: %d -> %d", len(pts), len(got))
	}
	// Endpoints preserved.
	if got[0] != (Vec{0, 0}) || got[len(got)-1] != (Vec{2, 2}) {
		t.Errorf("Chaikin moved endpoints: %v", got)
	}
	// The sharp corner at (2,0) should be rounded away (no point exactly there).
	for _, p := range got {
		if p == (Vec{2, 0}) {
			t.Error("Chaikin did not round the corner")
		}
	}
}

func TestSmoothStrokeReducesJitter(t *testing.T) {
	// A noisy near-straight stroke should end up with far fewer points and
	// stay close to the original line.
	var pts []Vec
	for i := 0; i <= 40; i++ {
		x := float64(i)
		y := 0.05 * math.Sin(float64(i)) // tiny jitter
		pts = append(pts, Vec{x, y})
	}
	got := SmoothStroke(pts, 0.5)
	if len(got) >= len(pts) {
		t.Errorf("SmoothStroke did not reduce points: %d -> %d", len(pts), len(got))
	}
	if got[0].Dist(pts[0]) > 0.001 {
		t.Error("start point drifted")
	}
}

func TestPolylineLength(t *testing.T) {
	pts := []Vec{{0, 0}, {3, 0}, {3, 4}}
	if l := PolylineLength(pts); math.Abs(l-7) > 1e-9 {
		t.Errorf("length = %f, want 7", l)
	}
}
