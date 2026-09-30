package media

import (
	"errors"
	"fmt"
	"image"
	"log"
	"math"
	"os"
	"strconv"
	"strings"

	"gocv.io/x/gocv"
)

// FaceRecognitionModel provides face embedding extraction for recognition
type FaceRecognitionModel struct {
	Net       gocv.Net
	Enabled   bool
	ModelName string

	// Configuration parameters
	InputSizeW  int
	InputSizeH  int
	ScaleFactor float64
	MeanVal     gocv.Scalar
	StdVal      gocv.Scalar
}

// NewFaceRecognitionModel loads a face recognition model (ArcFace, FaceNet, etc.)
func NewFaceRecognitionModel(modelPath string, modelName string) *FaceRecognitionModel {
	if modelPath == "" {
		log.Println("recognition: model path is empty, disabling face recognition")
		return &FaceRecognitionModel{Enabled: false}
	}

	log.Printf("recognition: Attempting to load %s model: %s", modelName, modelPath)

	// Check if file exists and is non-empty
	if info, err := os.Stat(modelPath); err != nil {
		if os.IsNotExist(err) {
			log.Printf("recognition: ERROR - Model file does not exist: %s", modelPath)
		} else {
			log.Printf("recognition: ERROR - Failed to stat model file %s: %v", modelPath, err)
		}
		return &FaceRecognitionModel{Enabled: false}
	} else if info.Size() == 0 {
		log.Printf("recognition: ERROR - Model file is empty (0 bytes): %s", modelPath)
		return &FaceRecognitionModel{Enabled: false}
	}

	var net gocv.Net
	lowerPath := strings.ToLower(modelPath)
	if strings.HasSuffix(lowerPath, ".onnx") {
		net = gocv.ReadNetFromONNX(modelPath)
	} else {
		net = gocv.ReadNet(modelPath, "")
	}
	if net.Empty() {
		log.Printf("recognition: ERROR - ReadNet returned an empty network for %s. Check file path and integrity.", modelName)
		return &FaceRecognitionModel{Enabled: false}
	}

	log.Printf("recognition: successfully loaded %s model", modelName)

	// OpenCV's CUDA backend silently produces empty output for ONNX models.
	// Default to CPU; set ARCFACE_CUDA=true to opt in.
	cudaEnabled := false
	if val := os.Getenv("ARCFACE_CUDA"); val != "" {
		if parsed, err := strconv.ParseBool(val); err == nil {
			cudaEnabled = parsed
		}
	}

	if cudaEnabled {
		cudaBackendErr := net.SetPreferableBackend(gocv.NetBackendCUDA)
		cudaTargetErr := net.SetPreferableTarget(gocv.NetTargetCUDA)
		if cudaBackendErr == nil && cudaTargetErr == nil {
			log.Printf("recognition: using CUDA for %s", modelName)
		} else {
			net.SetPreferableBackend(gocv.NetBackendDefault)
			net.SetPreferableTarget(gocv.NetTargetCPU)
			log.Printf("recognition: CUDA unavailable, using CPU for %s", modelName)
		}
	} else {
		net.SetPreferableBackend(gocv.NetBackendDefault)
		net.SetPreferableTarget(gocv.NetTargetCPU)
		log.Printf("recognition: using CPU for %s", modelName)
	}

	// Set model-specific parameters
	var inputSizeW, inputSizeH int
	var meanVal, stdVal gocv.Scalar

	switch modelName {
	case "arcface":
		inputSizeW, inputSizeH = 112, 112
		meanVal = gocv.NewScalar(127.5, 127.5, 127.5, 0)
		stdVal = gocv.NewScalar(128.0, 128.0, 128.0, 0)
	case "facenet":
		inputSizeW, inputSizeH = 160, 160
		meanVal = gocv.NewScalar(127.5, 127.5, 127.5, 0)
		stdVal = gocv.NewScalar(128.0, 128.0, 128.0, 0)
	default:
		inputSizeW, inputSizeH = 112, 112
		meanVal = gocv.NewScalar(127.5, 127.5, 127.5, 0)
		stdVal = gocv.NewScalar(128.0, 128.0, 128.0, 0)
	}

	return &FaceRecognitionModel{
		Net:         net,
		Enabled:     true,
		ModelName:   modelName,
		InputSizeW:  inputSizeW,
		InputSizeH:  inputSizeH,
		ScaleFactor: 1.0 / stdVal.Val1,
		MeanVal:     meanVal,
		StdVal:      stdVal,
	}
}

func (f *FaceRecognitionModel) Close() {
	if f != nil && f.Enabled {
		f.Net.Close()
		log.Printf("recognition: closed %s network", f.ModelName)
		f.Enabled = false
	}
}

// ExtractEmbedding extracts a face embedding from an unaligned face crop. Prefer
// ExtractEmbeddingAligned when landmarks are available.
func (f *FaceRecognitionModel) ExtractEmbedding(faceRegion gocv.Mat) ([]float32, error) {
	if f == nil || !f.Enabled {
		return nil, errors.New("face recognition model is not enabled")
	}
	if faceRegion.Empty() {
		return nil, errors.New("empty face region")
	}
	resized := gocv.NewMat()
	defer resized.Close()
	gocv.Resize(faceRegion, &resized, image.Pt(f.InputSizeW, f.InputSizeH), 0, 0, gocv.InterpolationLinear)
	return f.embed(resized)
}

// ExtractEmbeddingAligned warps the face in img onto the standard ArcFace
// template using its five landmarks before extracting the embedding.
func (f *FaceRecognitionModel) ExtractEmbeddingAligned(img gocv.Mat, landmarks []Point2D) ([]float32, error) {
	if f == nil || !f.Enabled {
		return nil, errors.New("face recognition model is not enabled")
	}
	if img.Empty() {
		return nil, errors.New("empty image")
	}
	aligned, err := alignFace(img, landmarks, f.InputSizeW)
	if err != nil {
		return nil, fmt.Errorf("face alignment failed: %w", err)
	}
	defer aligned.Close()
	return f.embed(aligned)
}

// embed runs the network on a BGR face image already at the model input size.
func (f *FaceRecognitionModel) embed(face gocv.Mat) ([]float32, error) {
	rgb := face
	if face.Channels() == 3 {
		rgb = gocv.NewMat()
		defer rgb.Close()
		gocv.CvtColor(face, &rgb, gocv.ColorBGRToRGB)
	}

	var blob gocv.Mat
	if f.ModelName == "arcface" {
		// arcface.onnx bakes (pixel - 127.5) * 0.0078125 normalization into the graph
		// itself (Sub/Mul nodes on the "data" input), so pass raw 0-255 pixel values
		// here rather than pre-scaling to 0-1 — otherwise the network's internal
		// subtraction wipes out nearly all of the input signal.
		blob = gocv.BlobFromImage(rgb, 1.0, image.Pt(f.InputSizeW, f.InputSizeH), gocv.NewScalar(0, 0, 0, 0), false, false)
	} else {
		// other models expect (pixel - mean) / std applied outside the graph
		blob = gocv.BlobFromImage(rgb, f.ScaleFactor, image.Pt(f.InputSizeW, f.InputSizeH), f.MeanVal, false, false)
	}
	defer blob.Close()

	f.Net.SetInput(blob, "")
	output := f.Net.Forward("")
	defer output.Close()

	embedding := f.extractEmbeddingVector(output)
	if len(embedding) == 0 {
		return nil, fmt.Errorf("empty embedding from model output shape %v", output.Size())
	}

	return f.normalizeEmbedding(embedding)
}

// extractEmbeddingVector extracts the embedding vector from model output
func (f *FaceRecognitionModel) extractEmbeddingVector(output gocv.Mat) []float32 {
	// Use Total() — Size() returns [] for 3D ONNX output blobs
	total := output.Total()
	if total == 0 {
		return nil
	}

	// Flatten to [1, N] so GetFloatAt(0, i) works
	flattened := output.Reshape(1, 1)
	defer flattened.Close()

	embeddingSize := flattened.Cols()
	if embeddingSize == 0 {
		embeddingSize = total
	}
	embedding := make([]float32, embeddingSize)
	for i := 0; i < embeddingSize; i++ {
		embedding[i] = flattened.GetFloatAt(0, i)
	}

	return embedding
}

// ErrZeroEmbedding is returned when the model produces an all-zero vector, which cannot be
// normalized and would yield NaN cosine distances in pgvector.
var ErrZeroEmbedding = errors.New("model produced a zero embedding")

// normalizeEmbedding normalizes the embedding vector to unit length.
func (f *FaceRecognitionModel) normalizeEmbedding(embedding []float32) ([]float32, error) {
	if len(embedding) == 0 {
		return nil, errors.New("empty embedding")
	}

	// Calculate L2 norm
	var norm float32
	for _, val := range embedding {
		norm += val * val
	}
	norm = float32(math.Sqrt(float64(norm)))

	if norm == 0 || math.IsNaN(float64(norm)) || math.IsInf(float64(norm), 0) {
		return nil, ErrZeroEmbedding
	}

	// Normalize
	normalized := make([]float32, len(embedding))
	for i, val := range embedding {
		normalized[i] = val / norm
	}

	return normalized, nil
}
