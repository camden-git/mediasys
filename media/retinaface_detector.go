package media

import (
	"errors"
	"fmt"
	"image"
	"log"
	"math"
	"os"
	"strconv"

	"gocv.io/x/gocv"
)

// RetinaFace prior box generation and box decoding utilities

// PriorBox defines an anchor box (center_x, center_y, width, height)
type PriorBox struct {
	Cx, Cy, W, H float32
}

// GenerateRetinaFacePriors generates priors for 640x640 RetinaFace
func GenerateRetinaFacePriors(imgW, imgH int) []PriorBox {
	// These settings match the standard RetinaFace/ONNX config
	minSizes := [][]int{{16, 32}, {64, 128}, {256, 512}}
	steps := []int{8, 16, 32}
	featureMapSizes := [][]int{
		{imgH / 8, imgW / 8},
		{imgH / 16, imgW / 16},
		{imgH / 32, imgW / 32},
	}
	priors := []PriorBox{}
	for k, fms := range featureMapSizes {
		fmH, fmW := fms[0], fms[1]
		for i := 0; i < fmH; i++ {
			for j := 0; j < fmW; j++ {
				for _, minSize := range minSizes[k] {
					cx := (float32(j) + 0.5) * float32(steps[k]) / float32(imgW)
					cy := (float32(i) + 0.5) * float32(steps[k]) / float32(imgH)
					w := float32(minSize) / float32(imgW)
					h := float32(minSize) / float32(imgH)
					priors = append(priors, PriorBox{Cx: cx, Cy: cy, W: w, H: h})
				}
			}
		}
	}
	return priors
}

// DecodeBox decodes a single box prediction using the prior and variances
func DecodeBox(rawBox [4]float32, prior PriorBox, variances [2]float32) [4]float32 {
	// rawBox: [dx, dy, dw, dh]
	cx := prior.Cx + rawBox[0]*variances[0]*prior.W
	cy := prior.Cy + rawBox[1]*variances[0]*prior.H
	w := prior.W * float32Exp(rawBox[2]*variances[1])
	h := prior.H * float32Exp(rawBox[3]*variances[1])
	// Convert center to corner
	x1 := cx - w/2
	y1 := cy - h/2
	x2 := cx + w/2
	y2 := cy + h/2
	return [4]float32{x1, y1, x2, y2}
}

// DecodeLandmarks decodes the 5 facial landmark predictions using the prior.
// Raw landmark values are offsets relative to the prior centre, scaled by variance[0].
func DecodeLandmarks(raw [10]float32, prior PriorBox, variance float32) [10]float32 {
	var out [10]float32
	for j := 0; j < 5; j++ {
		out[j*2] = prior.Cx + raw[j*2]*variance*prior.W
		out[j*2+1] = prior.Cy + raw[j*2+1]*variance*prior.H
	}
	return out
}

// float32Exp is a helper for float32 exponentiation
func float32Exp(x float32) float32 {
	return float32(math.Exp(float64(x)))
}

// RetinaFaceDetector provides high-accuracy face detection using RetinaFace
type RetinaFaceDetector struct {
	Net     gocv.Net
	Enabled bool

	// Configuration parameters
	InputSizeW    int
	InputSizeH    int
	ScaleFactor   float64
	MeanVal       gocv.Scalar
	ConfThreshold float32
	IoUThreshold  float32

	// Priors are pre-computed at construction time and reused across calls
	Priors []PriorBox
}

// NewRetinaFaceDetector loads the RetinaFace model
func NewRetinaFaceDetector(modelPath string) *RetinaFaceDetector {
	if modelPath == "" {
		log.Println("detection(retinaface): model path is empty, disabling RetinaFace detector")
		return &RetinaFaceDetector{Enabled: false}
	}

	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		log.Printf("detection(retinaface): model file not found: %s", modelPath)
		return &RetinaFaceDetector{Enabled: false}
	}

	log.Printf("detection(retinaface): Attempting to load model: %s", modelPath)

	net := gocv.ReadNetFromONNX(modelPath)
	if net.Empty() {
		log.Printf("detection(retinaface): ERROR - ReadNetFromONNX returned an empty network. Check file path and integrity.")
		return &RetinaFaceDetector{Enabled: false}
	}

	log.Printf("detection(retinaface): successfully loaded RetinaFace model")

	// OpenCV's CUDA backend has incomplete multi-output ONNX support — inference
	// silently produces empty mats. Use CPU for ONNX models; it is fast enough
	// for batch face detection. Set RETINAFACE_CUDA=true to opt in to CUDA.
	cudaEnabled := false
	if val := os.Getenv("RETINAFACE_CUDA"); val != "" {
		if parsed, err := strconv.ParseBool(val); err == nil {
			cudaEnabled = parsed
		}
	}

	if cudaEnabled {
		cudaBackendErr := net.SetPreferableBackend(gocv.NetBackendCUDA)
		cudaTargetErr := net.SetPreferableTarget(gocv.NetTargetCUDA)
		if cudaBackendErr == nil && cudaTargetErr == nil {
			log.Println("detection(retinaface): Set backend/target to CUDA")
		} else {
			net.SetPreferableBackend(gocv.NetBackendDefault)
			net.SetPreferableTarget(gocv.NetTargetCPU)
			log.Println("detection(retinaface): CUDA unavailable, using CPU")
		}
	} else {
		net.SetPreferableBackend(gocv.NetBackendDefault)
		net.SetPreferableTarget(gocv.NetTargetCPU)
		log.Println("detection(retinaface): using CPU backend")
	}

	return &RetinaFaceDetector{
		Net:           net,
		Enabled:       true,
		InputSizeW:    640,
		InputSizeH:    640,
		ScaleFactor:   1.0,
		MeanVal:       gocv.NewScalar(104.0, 117.0, 123.0, 0),
		ConfThreshold: 0.5,
		IoUThreshold:  0.5,
		Priors:        GenerateRetinaFacePriors(640, 640),
	}
}

func (r *RetinaFaceDetector) Close() {
	if r != nil && r.Enabled {
		r.Net.Close()
		log.Println("detection(retinaface): closed network")
		r.Enabled = false
	}
}

// letterbox scales img to fit the network input while keeping its aspect ratio,
// anchored top-left and padded with the mean colour (which becomes zero after
// mean subtraction). It returns the canvas and the applied scale factor.
func (r *RetinaFaceDetector) letterbox(img gocv.Mat) (gocv.Mat, float32) {
	scale := min(float32(r.InputSizeW)/float32(img.Cols()), float32(r.InputSizeH)/float32(img.Rows()))
	newW := max(1, min(r.InputSizeW, int(float32(img.Cols())*scale+0.5)))
	newH := max(1, min(r.InputSizeH, int(float32(img.Rows())*scale+0.5)))

	interp := gocv.InterpolationLinear
	if scale < 1 {
		interp = gocv.InterpolationArea
	}
	resized := gocv.NewMat()
	defer resized.Close()
	gocv.Resize(img, &resized, image.Pt(newW, newH), 0, 0, interp)

	canvas := gocv.NewMatWithSizeFromScalar(r.MeanVal, r.InputSizeH, r.InputSizeW, resized.Type())
	roi := canvas.Region(image.Rect(0, 0, newW, newH))
	resized.CopyTo(&roi)
	roi.Close()
	return canvas, scale
}

// DetectFaces runs face detection using RetinaFace
func (r *RetinaFaceDetector) DetectFaces(img gocv.Mat) ([]DetectionResult, error) {
	if r == nil || !r.Enabled {
		return nil, errors.New("retinaface detector is not enabled")
	}
	if img.Empty() {
		return nil, errors.New("empty image")
	}

	imgHeight := float32(img.Rows())
	imgWidth := float32(img.Cols())

	canvas, scale := r.letterbox(img)
	defer canvas.Close()

	blob := gocv.BlobFromImage(canvas, 1.0, image.Pt(r.InputSizeW, r.InputSizeH), r.MeanVal, false, false)
	defer blob.Close()

	r.Net.SetInput(blob, "")

	// Discover the actual output layer names at runtime rather than hardcoding them.
	allNames := r.Net.GetLayerNames()
	outIDs := r.Net.GetUnconnectedOutLayers()
	outputNames := make([]string, 0, len(outIDs))
	for _, id := range outIDs {
		if id >= 1 && id <= len(allNames) {
			outputNames = append(outputNames, allNames[id-1])
		}
	}
	if len(outputNames) < 3 {
		return nil, fmt.Errorf("retinaface model has %d output layers (need at least 3): %v", len(outputNames), outputNames)
	}

	// Fetch all outputs in one pass: separate Forward(name) calls reuse internal
	// buffers, so earlier outputs get overwritten and detections come out garbage.
	mats := r.Net.ForwardLayers(outputNames)
	defer func() {
		for i := range mats {
			mats[i].Close()
		}
	}()

	// Match each output to boxes / scores / landmarks by total element count.
	// Size() doesn't work for 3D ONNX blobs in gocv; Total() always works.
	// For 640×640 RetinaFace ResNet50: 16800 priors.
	//   boxes:     16800 × 4  = 67200 elements
	//   scores:    16800 × 2  = 33600 elements
	//   landmarks: 16800 × 10 = 168000 elements
	numPriors := len(r.Priors)

	var boxesMat, scoresMat, landmarksMat *gocv.Mat
	for i := range mats {
		switch mats[i].Total() {
		case numPriors * 4:
			boxesMat = &mats[i]
		case numPriors * 2:
			scoresMat = &mats[i]
		case numPriors * 10:
			landmarksMat = &mats[i]
		}
	}

	if boxesMat == nil || scoresMat == nil || landmarksMat == nil {
		totals := make([]int, len(mats))
		for i, m := range mats {
			totals[i] = m.Total()
		}
		return nil, fmt.Errorf("retinaface: could not identify outputs (numPriors=%d, totals=%v, names=%v)", numPriors, totals, outputNames)
	}

	return r.parseRetinaFaceOutput(*boxesMat, *scoresMat, *landmarksMat, imgWidth, imgHeight, scale)
}

// parseRetinaFaceOutput parses the RetinaFace model outputs (boxes, scores, landmarks)
func (r *RetinaFaceDetector) parseRetinaFaceOutput(boxes, scores, landmarks gocv.Mat, imgWidth, imgHeight, scale float32) ([]DetectionResult, error) {
	// Derive numDetections from total element count (Total() works; Size()[1] doesn't for 3D blobs).
	// boxes has numPriors×4 elements → numDetections = Total()/4
	numDetections := boxes.Total() / 4

	// Predictions are normalised to the (letterboxed) network input, so
	// multiplying by inverse maps them back to original image pixels.
	inverse := float32(r.InputSizeW) / scale

	priors := r.Priors
	if len(priors) != numDetections {
		return nil, fmt.Errorf("retinaface: prior count %d != numDetections %d", len(priors), numDetections)
	}

	// Flatten each 3-D tensor [1, N, K] → [N, K] so GetFloatAt works as [row, col].
	boxes2D := boxes.Reshape(1, numDetections)
	defer boxes2D.Close()
	scores2D := scores.Reshape(1, numDetections)
	defer scores2D.Close()
	landmarks2D := landmarks.Reshape(1, numDetections)
	defer landmarks2D.Close()
	variances := [2]float32{0.1, 0.2}

	var detections []DetectionResult
	for i := 0; i < numDetections; i++ {
		// Column 1 of the 2-class softmax output is the face score.
		scoreFace := scores2D.GetFloatAt(i, 1)
		if scoreFace < r.ConfThreshold {
			continue
		}

		var rawBox [4]float32
		for j := 0; j < 4; j++ {
			rawBox[j] = boxes2D.GetFloatAt(i, j)
		}
		decoded := DecodeBox(rawBox, priors[i], variances)
		x1 := max(0, decoded[0]*inverse)
		y1 := max(0, decoded[1]*inverse)
		x2 := min(imgWidth, decoded[2]*inverse)
		y2 := min(imgHeight, decoded[3]*inverse)
		if x2 <= x1 || y2 <= y1 {
			continue
		}

		var rawLM [10]float32
		for j := 0; j < 10; j++ {
			rawLM[j] = landmarks2D.GetFloatAt(i, j)
		}
		decodedLM := DecodeLandmarks(rawLM, priors[i], variances[0])
		var pts []Point2D
		for j := 0; j < 5; j++ {
			pts = append(pts, Point2D{
				X: decodedLM[j*2] * inverse,
				Y: decodedLM[j*2+1] * inverse,
			})
		}

		if !validateLandmarkGeometry(pts, x1, y1, x2, y2) {
			continue
		}

		faceArea := (x2 - x1) * (y2 - y1)
		qs := scoreFace * (faceArea / (imgWidth * imgHeight)) * 100
		detections = append(detections, DetectionResult{
			X:            int(x1),
			Y:            int(y1),
			W:            int(x2 - x1),
			H:            int(y2 - y1),
			Confidence:   scoreFace,
			Landmarks:    pts,
			ModelName:    "retinaface",
			QualityScore: &qs,
		})
	}

	detections = r.nonMaxSuppression(detections)
	return detections, nil
}

// validateLandmarkGeometry filters phantom detections that span two nearby faces.
// RetinaFace landmarks: [0]=left eye, [1]=right eye, [2]=nose, [3]=left mouth, [4]=right mouth.
//
// Two checks catch the "one eye from each person" case regardless of box shape:
//  1. Inter-eye distance / box width > 0.65 → eyes are near opposite edges, not a single face.
//  2. |nose_x - eye_midpoint_x| / box_width > 0.20 → nose isn't centred between the eyes.
func validateLandmarkGeometry(pts []Point2D, x1, y1, x2, y2 float32) bool {
	if len(pts) < 3 {
		return true // no landmarks to check, let NMS handle it
	}

	boxW := x2 - x1
	if boxW <= 0 {
		return false
	}

	leftEye := pts[0]
	rightEye := pts[1]
	nose := pts[2]

	// 1. Inter-eye distance should not span most of the box width.
	interEye := rightEye.X - leftEye.X
	if interEye < 0 {
		interEye = -interEye
	}
	if interEye/boxW > 0.65 {
		log.Printf("detection(retinaface): dropping phantom detection — inter-eye ratio %.2f", interEye/boxW)
		return false
	}

	// 2. Nose should be roughly centred between the two eyes horizontally.
	eyeMidX := (leftEye.X + rightEye.X) / 2
	noseDeviation := nose.X - eyeMidX
	if noseDeviation < 0 {
		noseDeviation = -noseDeviation
	}
	if noseDeviation/boxW > 0.20 {
		log.Printf("detection(retinaface): dropping phantom detection — nose deviation ratio %.2f", noseDeviation/boxW)
		return false
	}

	return true
}

// nonMaxSuppression applies NMS to remove overlapping detections
func (r *RetinaFaceDetector) nonMaxSuppression(detections []DetectionResult) []DetectionResult {
	if len(detections) == 0 {
		return detections
	}

	// Sort by confidence (highest first)
	for i := 0; i < len(detections)-1; i++ {
		for j := i + 1; j < len(detections); j++ {
			if detections[i].Confidence < detections[j].Confidence {
				detections[i], detections[j] = detections[j], detections[i]
			}
		}
	}

	// Apply NMS
	var result []DetectionResult
	used := make([]bool, len(detections))

	for i := 0; i < len(detections); i++ {
		if used[i] {
			continue
		}

		result = append(result, detections[i])
		used[i] = true

		for j := i + 1; j < len(detections); j++ {
			if used[j] {
				continue
			}

			// Calculate IoU
			iou := r.calculateIoU(detections[i], detections[j])
			if iou > r.IoUThreshold {
				used[j] = true
			}
		}
	}

	return result
}

// calculateIoU calculates the Intersection over Union between two detections
func (r *RetinaFaceDetector) calculateIoU(a, b DetectionResult) float32 {
	// Calculate intersection rectangle
	x1 := max(a.X, b.X)
	y1 := max(a.Y, b.Y)
	x2 := min(a.X+a.W, b.X+b.W)
	y2 := min(a.Y+a.H, b.Y+b.H)

	if x2 <= x1 || y2 <= y1 {
		return 0.0
	}

	intersection := float32((x2 - x1) * (y2 - y1))
	areaA := float32(a.W * a.H)
	areaB := float32(b.W * b.H)
	union := areaA + areaB - intersection

	return intersection / union
}

// DetectFacesAndExtractEmbeddings detects faces and extracts embeddings. When a
// recognition model is enabled, failing to embed any face is an error so faces
// are never stored without an embedding.
func (r *RetinaFaceDetector) DetectFacesAndExtractEmbeddings(img gocv.Mat, recognitionModel *FaceRecognitionModel) ([]DetectionResult, error) {
	detections, err := r.DetectFaces(img)
	if err != nil {
		return nil, err
	}

	if recognitionModel != nil && recognitionModel.Enabled {
		for i := range detections {
			faceRegion := img.Region(image.Rect(
				detections[i].X, detections[i].Y,
				detections[i].X+detections[i].W, detections[i].Y+detections[i].H,
			))
			embedding, err := recognitionModel.ExtractEmbedding(faceRegion)
			faceRegion.Close()
			if err != nil {
				return nil, fmt.Errorf("face %d: %w", i, err)
			}
			detections[i].Embedding = embedding
			detections[i].ModelName = recognitionModel.ModelName
		}
	}

	return detections, nil
}
