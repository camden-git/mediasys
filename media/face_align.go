package media

import (
	"errors"
	"image"

	"gocv.io/x/gocv"
)

// arcFaceTemplate is the standard ArcFace 112x112 landmark template: left eye,
// right eye, nose tip, left mouth corner, right mouth corner.
var arcFaceTemplate = [5]Point2D{
	{X: 38.2946, Y: 51.6963},
	{X: 73.5318, Y: 51.5014},
	{X: 56.0252, Y: 71.7366},
	{X: 41.5493, Y: 92.3655},
	{X: 70.7299, Y: 92.2041},
}

// arcFaceTemplateSize is the side length the template coordinates refer to.
const arcFaceTemplateSize = 112

// similarityTransform returns the least-squares similarity transform
// (rotation, uniform scale, translation) mapping src onto dst as a row-major
// 2x3 matrix [a -b tx; b a ty].
func similarityTransform(src, dst []Point2D) ([6]float64, error) {
	var m [6]float64
	if len(src) != len(dst) || len(src) < 2 {
		return m, errors.New("similarity transform needs at least two matching points")
	}
	n := float64(len(src))
	var spx, spy, sqx, sqy float64
	for i := range src {
		spx += float64(src[i].X)
		spy += float64(src[i].Y)
		sqx += float64(dst[i].X)
		sqy += float64(dst[i].Y)
	}
	pmx, pmy, qmx, qmy := spx/n, spy/n, sqx/n, sqy/n

	var num1, num2, den float64
	for i := range src {
		px, py := float64(src[i].X)-pmx, float64(src[i].Y)-pmy
		qx, qy := float64(dst[i].X)-qmx, float64(dst[i].Y)-qmy
		num1 += px*qx + py*qy
		num2 += px*qy - py*qx
		den += px*px + py*py
	}
	if den == 0 {
		return m, errors.New("degenerate landmarks")
	}
	a, b := num1/den, num2/den
	m = [6]float64{a, -b, qmx - (a*pmx - b*pmy), b, a, qmy - (b*pmx + a*pmy)}
	return m, nil
}

// alignFace warps img so the five landmarks land on the ArcFace template,
// producing a size x size crop.
func alignFace(img gocv.Mat, landmarks []Point2D, size int) (gocv.Mat, error) {
	if len(landmarks) != len(arcFaceTemplate) {
		return gocv.Mat{}, errors.New("alignment needs 5 landmarks")
	}
	ratio := float32(size) / arcFaceTemplateSize
	dst := make([]Point2D, len(arcFaceTemplate))
	for i, p := range arcFaceTemplate {
		dst[i] = Point2D{X: p.X * ratio, Y: p.Y * ratio}
	}
	m, err := similarityTransform(landmarks, dst)
	if err != nil {
		return gocv.Mat{}, err
	}

	warp := gocv.NewMatWithSize(2, 3, gocv.MatTypeCV64F)
	defer warp.Close()
	for i, v := range m {
		warp.SetDoubleAt(i/3, i%3, v)
	}

	out := gocv.NewMat()
	gocv.WarpAffine(img, &out, warp, image.Pt(size, size))
	return out, nil
}
