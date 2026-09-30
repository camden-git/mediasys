package media

import (
	"math"
	"testing"
)

func rotatePoints(pts []Point2D, angle float64) []Point2D {
	sin, cos := math.Sincos(angle)
	out := make([]Point2D, len(pts))
	for i, p := range pts {
		out[i] = Point2D{
			X: float32(cos*float64(p.X) - sin*float64(p.Y)),
			Y: float32(sin*float64(p.X) + cos*float64(p.Y)),
		}
	}
	return out
}

func boundsOf(pts []Point2D, pad float32) (x1, y1, x2, y2 float32) {
	x1, y1, x2, y2 = pts[0].X, pts[0].Y, pts[0].X, pts[0].Y
	for _, p := range pts {
		x1, y1 = min(x1, p.X), min(y1, p.Y)
		x2, y2 = max(x2, p.X), max(y2, p.Y)
	}
	return x1 - pad, y1 - pad, x2 + pad, y2 + pad
}

func TestValidateLandmarkGeometry(t *testing.T) {
	face := []Point2D{{X: 38, Y: 52}, {X: 74, Y: 52}, {X: 56, Y: 72}, {X: 42, Y: 92}, {X: 71, Y: 92}}
	x1, y1, x2, y2 := boundsOf(face, 20)
	if !validateLandmarkGeometry(face, x1, y1, x2, y2) {
		t.Fatal("upright face rejected")
	}

	for _, deg := range []float64{30, 60, 90, 135, 180} {
		rolled := rotatePoints(face, deg*math.Pi/180)
		x1, y1, x2, y2 := boundsOf(rolled, 20)
		if !validateLandmarkGeometry(rolled, x1, y1, x2, y2) {
			t.Errorf("face rolled by %v degrees rejected", deg)
		}
	}

	// one eye from each of two people: eyes span nearly the whole box
	phantom := []Point2D{{X: 0, Y: 50}, {X: 100, Y: 50}, {X: 50, Y: 70}}
	if validateLandmarkGeometry(phantom, -5, 30, 105, 120) {
		t.Error("phantom detection accepted")
	}

	// nose far off to the side of the eyes
	skewed := []Point2D{{X: 38, Y: 52}, {X: 74, Y: 52}, {X: 130, Y: 72}}
	if validateLandmarkGeometry(skewed, 0, 20, 140, 120) {
		t.Error("off-centre nose accepted")
	}
}

func TestSimilarityTransformMapsTemplate(t *testing.T) {
	src := rotatePoints(arcFaceTemplate[:], 0.4)
	for i := range src {
		src[i].X = src[i].X*2 + 300
		src[i].Y = src[i].Y*2 + 120
	}
	m, err := similarityTransform(src, arcFaceTemplate[:])
	if err != nil {
		t.Fatal(err)
	}
	for i, p := range src {
		x := m[0]*float64(p.X) + m[1]*float64(p.Y) + m[2]
		y := m[3]*float64(p.X) + m[4]*float64(p.Y) + m[5]
		if math.Abs(x-float64(arcFaceTemplate[i].X)) > 0.01 || math.Abs(y-float64(arcFaceTemplate[i].Y)) > 0.01 {
			t.Errorf("point %d maps to (%.3f, %.3f), want (%.3f, %.3f)", i, x, y, arcFaceTemplate[i].X, arcFaceTemplate[i].Y)
		}
	}
}
