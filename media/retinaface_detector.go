package media

import (
	"image"
	"log"
	"math"

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
}

// NewRetinaFaceDetector loads the RetinaFace model
func NewRetinaFaceDetector(modelPath string) *RetinaFaceDetector {
	if modelPath == "" {
		log.Println("detection(retinaface): model path is empty, disabling RetinaFace detector")
		return &RetinaFaceDetector{Enabled: false}
	}

	log.Printf("detection(retinaface): Attempting to load model: %s", modelPath)

	net := gocv.ReadNet(modelPath, "")
	if net.Empty() {
		log.Printf("detection(retinaface): ERROR - ReadNet returned an empty network. Check file path and integrity.")
		return &RetinaFaceDetector{Enabled: false}
	}

	log.Printf("detection(retinaface): successfully loaded RetinaFace model")

	// Try to use CUDA if available
	cudaBackendErr := net.SetPreferableBackend(gocv.NetBackendCUDA)
	cudaTargetErr := net.SetPreferableTarget(gocv.NetTargetCUDA)

	if cudaBackendErr == nil && cudaTargetErr == nil {
		log.Println("detection(retinaface): Set backend/target to CUDA")
	} else {
		if cudaBackendErr != nil {
			log.Printf("detection(retinaface): CUDA Backend not available: %v. Using default backend.", cudaBackendErr)
		}
		if cudaTargetErr != nil {
			log.Printf("detection(retinaface): CUDA Target not available: %v. Using default target.", cudaTargetErr)
		}

		net.SetPreferableBackend(gocv.NetBackendDefault)
		net.SetPreferableTarget(gocv.NetTargetCPU)
		log.Println("detection(retinaface): Set backend/target to CPU (Default)")
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
	}
}

func (r *RetinaFaceDetector) Close() {
	if r != nil && r.Enabled {
		r.Net.Close()
		log.Println("detection(retinaface): closed network")
		r.Enabled = false
	}
}

// DetectFaces runs face detection using RetinaFace
func (r *RetinaFaceDetector) DetectFaces(img gocv.Mat) []DetectionResult {
	if r == nil || !r.Enabled || img.Empty() {
		return nil
	}

	imgHeight := float32(img.Rows())
	imgWidth := float32(img.Cols())

	// Manually convert to RGB before blob creation
	blob := gocv.BlobFromImage(img, 1.0, image.Pt(r.InputSizeW, r.InputSizeH), gocv.NewScalar(104.0, 117.0, 123.0, 0), false, false)
	defer blob.Close()

	r.Net.SetInput(blob, "input")

	// Use output names as seen in the log: bbox, confidence, landmark
	outputNames := []string{"bbox", "confidence", "landmark"}
	outputs := r.Net.ForwardLayers(outputNames)
	if len(outputs) < 3 {
		log.Printf("detection(retinaface): Expected 3 outputs (boxes, scores, landmarks), got %d", len(outputs))
		return nil
	}
	// Debug: print output shapes and first few values
	for idx, out := range outputs {
		shape := out.Size()
		log.Printf("detection(retinaface): Output %d shape: %v", idx, shape)
		// Print first 10 values (flattened)
		flat := out.Reshape(1, 1)
		vals := []float32{}
		for i := 0; i < flat.Cols() && i < 10; i++ {
			vals = append(vals, flat.GetFloatAt(0, i))
		}
		flat.Close()
		log.Printf("detection(retinaface): Output %d first values: %v", idx, vals)
	}
	defer func() {
		for _, mat := range outputs {
			mat.Close()
		}
	}()
	boxes := outputs[0]
	scores := outputs[1]
	landmarks := outputs[2]
	return r.parseRetinaFaceOutput(boxes, scores, landmarks, imgWidth, imgHeight)
}

// parseRetinaFaceOutput parses the RetinaFace model outputs (boxes, scores, landmarks)
func (r *RetinaFaceDetector) parseRetinaFaceOutput(boxes, scores, landmarks gocv.Mat, imgWidth, imgHeight float32) []DetectionResult {
	var detections []DetectionResult

	// Debug: Print tensor shapes
	boxesShape := boxes.Size()
	scoresShape := scores.Size()
	landmarksShape := landmarks.Size()
	log.Printf("detection(retinaface): Debug - Boxes shape: %v, Scores shape: %v, Landmarks shape: %v", boxesShape, scoresShape, landmarksShape)

	// All outputs are [1, N, ...], so get N
	numDetections := boxes.Size()[1]
	log.Printf("detection(retinaface): Debug - Processing %d detections", numDetections)

	// Generate priors for 640x640
	priors := GenerateRetinaFacePriors(640, 640)
	if len(priors) != numDetections {
		log.Printf("detection(retinaface): WARNING - priors count (%d) != numDetections (%d)", len(priors), numDetections)
		return nil
	}
	variances := [2]float32{0.1, 0.2}

	// Debug: Check scores above different thresholds
	thresholds := []float32{0.1, 0.3, 0.5, 0.7, 0.9}
	for _, threshold := range thresholds {
		count := 0
		for i := 0; i < numDetections; i++ {
			scoreFace := scores.GetFloatAt(0, i*2+1)
			if scoreFace > threshold {
				count++
			}
		}
		log.Printf("detection(retinaface): Debug - Scores > %.1f: %d detections", threshold, count)
	}

	// Debug: Print first 10 scores and their corresponding decoded boxes
	log.Printf("detection(retinaface): Debug - First 10 detections (score, DECODED box coordinates):")
	for i := 0; i < minInt(10, numDetections); i++ {
		scoreFace := scores.GetFloatAt(0, i*2+1)
		// Get raw box
		var rawBox [4]float32
		for j := 0; j < 4; j++ {
			rawBox[j] = boxes.GetFloatAt(0, i*4+j)
		}
		decoded := DecodeBox(rawBox, priors[i], variances)
		log.Printf("detection(retinaface): Debug - Detection %d: score=%.4f, decoded_box=[%.3f,%.3f,%.3f,%.3f]",
			i, scoreFace, decoded[0], decoded[1], decoded[2], decoded[3])
	}

	// Now process with lower threshold for debugging
	debugThreshold := float32(0.1) // Show ALL detections for debugging
	log.Printf("detection(retinaface): Debug - Using threshold %.3f for debugging", debugThreshold)

	for i := 0; i < numDetections; i++ {
		scoreFace := scores.GetFloatAt(0, i*2+1)
		if scoreFace < debugThreshold {
			continue
		}
		// Get and decode box
		var rawBox [4]float32
		for j := 0; j < 4; j++ {
			rawBox[j] = boxes.GetFloatAt(0, i*4+j)
		}
		decoded := DecodeBox(rawBox, priors[i], variances)
		x1 := decoded[0] * imgWidth
		y1 := decoded[1] * imgHeight
		x2 := decoded[2] * imgWidth
		y2 := decoded[3] * imgHeight
		// Clamp to image boundaries
		x1 = maxFloat32(0, x1)
		y1 = maxFloat32(0, y1)
		x2 = minFloat32(imgWidth, x2)
		y2 = minFloat32(imgHeight, y2)
		if x2 <= x1 || y2 <= y1 {
			if scoreFace > 0.5 {
				log.Printf("detection(retinaface): Debug - Invalid decoded box for detection %d: [%.1f,%.1f,%.1f,%.1f]",
					i, x1, y1, x2, y2)
			}
			continue
		}
		// Landmarks (5 points, still need to decode if model outputs encoded landmarks)
		var pts []Point2D
		for j := 0; j < 5; j++ {
			lx := landmarks.GetFloatAt(0, i*10+j*2+0) * imgWidth
			ly := landmarks.GetFloatAt(0, i*10+j*2+1) * imgHeight
			pts = append(pts, Point2D{X: lx, Y: ly})
		}
		faceArea := float32((x2 - x1) * (y2 - y1))
		imageArea := imgWidth * imgHeight
		relativeSize := faceArea / imageArea
		qualityScore := scoreFace * relativeSize * 100
		detection := DetectionResult{
			X:            int(x1),
			Y:            int(y1),
			W:            int(x2 - x1),
			H:            int(y2 - y1),
			Confidence:   scoreFace,
			Landmarks:    pts,
			ModelName:    "retinaface",
			QualityScore: &qualityScore,
		}
		detections = append(detections, detection)
		if scoreFace > 0.5 {
			log.Printf("detection(retinaface): Debug - Added detection %d: score=%.4f, box=[%d,%d,%d,%d], area=%.1f",
				i, scoreFace, detection.X, detection.Y, detection.W, detection.H, faceArea)
		}
	}

	log.Printf("detection(retinaface): Parsed %d valid detections (with debug threshold %.3f)", len(detections), debugThreshold)

	// Now filter by actual confidence threshold
	var finalDetections []DetectionResult
	for _, det := range detections {
		if det.Confidence >= r.ConfThreshold {
			finalDetections = append(finalDetections, det)
		}
	}

	log.Printf("detection(retinaface): Final detections after confidence threshold %.3f: %d", r.ConfThreshold, len(finalDetections))

	// Apply Non-Maximum Suppression to remove overlapping detections
	finalDetections = r.nonMaxSuppression(finalDetections)
	log.Printf("detection(retinaface): Final detections after NMS: %d", len(finalDetections))

	return finalDetections
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
	x1 := maxInt(a.X, b.X)
	y1 := maxInt(a.Y, b.Y)
	x2 := minInt(a.X+a.W, b.X+b.W)
	y2 := minInt(a.Y+a.H, b.Y+b.H)

	if x2 <= x1 || y2 <= y1 {
		return 0.0
	}

	intersection := float32((x2 - x1) * (y2 - y1))
	areaA := float32(a.W * a.H)
	areaB := float32(b.W * b.H)
	union := areaA + areaB - intersection

	return intersection / union
}

// DetectFacesAndExtractEmbeddings detects faces and extracts embeddings
func (r *RetinaFaceDetector) DetectFacesAndExtractEmbeddings(img gocv.Mat, recognitionModel *FaceRecognitionModel) []DetectionResult {
	detections := r.DetectFaces(img)
	log.Printf("detection(retinaface): Found %d faces, recognition model enabled: %v", len(detections), recognitionModel != nil && recognitionModel.Enabled)

	if recognitionModel != nil && recognitionModel.Enabled {
		for i := range detections {
			// Extract face region
			faceRegion := img.Region(image.Rect(detections[i].X, detections[i].Y,
				detections[i].X+detections[i].W, detections[i].Y+detections[i].H))

			// DEBUG: Save the crop for face 480 in topgolf17/topgolf17-60.jpg
			if detections[i].X == 1838 && detections[i].Y == 1005 && detections[i].W == 1368 && detections[i].H == 1881 {
				// Save the crop as JPEG
				gocv.IMWrite("face_crop_480.jpg", faceRegion)
				log.Printf("DEBUG: Saved face crop for face 480 as face_crop_480.jpg")
			}

			log.Printf("detection(retinaface): Extracting embedding for face %d at [%d,%d,%d,%d]", i, detections[i].X, detections[i].Y, detections[i].W, detections[i].H)

			// Extract embedding
			embedding := recognitionModel.ExtractEmbedding(faceRegion)
			if embedding != nil {
				detections[i].Embedding = embedding
				detections[i].ModelName = recognitionModel.ModelName
				log.Printf("detection(retinaface): Successfully extracted embedding of length %d for face %d", len(embedding), i)
			} else {
				log.Printf("detection(retinaface): WARNING - Failed to extract embedding for face %d", i)
			}
		}
	}

	return detections
}
