package utils

import (
	"fmt"
	"image"
	"log"

	"gocv.io/x/gocv"
)

type DetectionResult struct {
	X          int
	Y          int
	W          int
	H          int
	Confidence float32
}

type DNNFaceDetector struct {
	Net     gocv.Net
	Enabled bool

	// configuration parameters used during detection
	InputSizeW    int
	InputSizeH    int
	ScaleFactor   float64
	MeanVal       gocv.Scalar
	ConfThreshold float32
}

// NewDNNFaceDetector loads the DNN model
func NewDNNFaceDetector(configPath, modelPath string) *DNNFaceDetector {
	if configPath == "" || modelPath == "" {
		log.Println("detection(dnn): config or model path is empty, disabling DNN detector")
		return &DNNFaceDetector{Enabled: false}
	}

	net := gocv.ReadNet(modelPath, configPath)
	if net.Empty() {
		log.Printf("detection(dnn): ERROR loading network model: config=%s, model=%s", configPath, modelPath)
		return &DNNFaceDetector{Enabled: false}
	}
	log.Printf("detection(dnn): successfully loaded face detection model")

	cudaBackendErr := net.SetPreferableBackend(gocv.NetBackendCUDA)
	cudaTargetErr := net.SetPreferableTarget(gocv.NetTargetCUDA)

	if cudaBackendErr == nil && cudaTargetErr == nil {
		log.Println("detection(dnn): Set backend/target to CUDA")
	} else {
		if cudaBackendErr != nil {
			log.Printf("detection(dnn): CUDA Backend not available or failed: %v. Using default backend.", cudaBackendErr)
		}
		if cudaTargetErr != nil {
			log.Printf("detection(dnn): CUDA Target not available or failed: %v. Using default target.", cudaTargetErr)
		}

		net.SetPreferableBackend(gocv.NetBackendDefault) // or gocv.NetBackendOpenCV
		net.SetPreferableTarget(gocv.NetTargetCPU)
		log.Println("detection(dnn): Set backend/target to CPU (Default)")
	}

	return &DNNFaceDetector{
		Net:           net,
		Enabled:       true,
		InputSizeW:    300,
		InputSizeH:    300,
		ScaleFactor:   1.0,
		MeanVal:       gocv.NewScalar(104.0, 177.0, 123.0, 0),
		ConfThreshold: 0.2,
	}
}

func (d *DNNFaceDetector) Close() {
	if d != nil && d.Enabled {
		d.Net.Close()
		log.Println("detection(dnn): closed network")
		d.Enabled = false
	}
}

// DetectFaces runs face detection using the loaded DNN model
func (d *DNNFaceDetector) DetectFaces(img gocv.Mat) []DetectionResult {
	if d == nil || !d.Enabled || img.Empty() {
		return nil
	}

	imgHeight := float32(img.Rows())
	imgWidth := float32(img.Cols())

	blob := gocv.BlobFromImage(img, d.ScaleFactor, image.Pt(d.InputSizeW, d.InputSizeH), d.MeanVal, false, false)
	defer blob.Close()

	d.Net.SetInput(blob, "")
	detectionsMat := d.Net.Forward("")
	defer detectionsMat.Close()

	results := []DetectionResult{}

	sizes := detectionsMat.Size()
	if len(sizes) != 4 || sizes[0] != 1 || sizes[1] != 1 {
		log.Printf("detection(dnn): Warning - Unexpected output matrix dimensions: %v", sizes)

		if len(sizes) < 3 {
			log.Printf("detection(dnn): Error - Output matrix dimensions too small to parse")
			return results
		}
	}

	numDetections := sizes[2]
	if numDetections == 0 {
		// log.Printf("detection(dnn): No detections in output matrix.")
		return results // No detections found
	}

	// reshape the Mat to 2D: [N, 7] for easier access with GetFloatAt(row, col)
	detections2D := detectionsMat.Reshape(1, numDetections*sizes[3])
	detectionsData := detections2D.Reshape(1, numDetections)
	defer detectionsData.Close()

	for i := 0; i < numDetections; i++ {
		confidence := detectionsData.GetFloatAt(i, 2)

		if confidence > d.ConfThreshold {
			xMin := detectionsData.GetFloatAt(i, 3) * imgWidth
			yMin := detectionsData.GetFloatAt(i, 4) * imgHeight
			xMax := detectionsData.GetFloatAt(i, 5) * imgWidth
			yMax := detectionsData.GetFloatAt(i, 6) * imgHeight

			xMin = max(0, xMin)
			yMin = max(0, yMin)
			xMax = min(imgWidth, xMax)
			yMax = min(imgHeight, yMax)

			if xMax > xMin && yMax > yMin {
				results = append(results, DetectionResult{
					X:          int(xMin),
					Y:          int(yMin),
					W:          int(xMax - xMin),
					H:          int(yMax - yMin),
					Confidence: confidence,
				})
			}
		}
	}

	return results
}

func DetectFacesAndAnimals(imagePath string, faceDetector *DNNFaceDetector) ([]DetectionResult, error) {
	if faceDetector == nil || !faceDetector.Enabled {
		log.Println("detection(dnn): face detector not provided or not enabled")
		return nil, nil
	}

	img := gocv.IMRead(imagePath, gocv.IMReadColor)
	if img.Empty() {
		return nil, fmt.Errorf("failed to read image file for dnn: %s", imagePath)
	}
	defer img.Close()

	detections := faceDetector.DetectFaces(img)
	log.Printf("detection(dnn): found %d face(s) in %s", len(detections), imagePath)

	return detections, nil
}
