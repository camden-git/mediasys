package media

import (
	"image"
	"log"
	"math"
	"os"

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

	// Check if file exists
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		log.Printf("recognition: ERROR - Model file does not exist: %s", modelPath)
		return &FaceRecognitionModel{Enabled: false}
	}

	net := gocv.ReadNet(modelPath, "")
	if net.Empty() {
		log.Printf("recognition: ERROR - ReadNet returned an empty network for %s. Check file path and integrity.", modelName)
		return &FaceRecognitionModel{Enabled: false}
	}

	log.Printf("recognition: successfully loaded %s model", modelName)

	// Try to use CUDA if available
	cudaBackendErr := net.SetPreferableBackend(gocv.NetBackendCUDA)
	cudaTargetErr := net.SetPreferableTarget(gocv.NetTargetCUDA)

	if cudaBackendErr == nil && cudaTargetErr == nil {
		log.Printf("recognition: Set backend/target to CUDA for %s", modelName)
	} else {
		if cudaBackendErr != nil {
			log.Printf("recognition: CUDA Backend not available for %s: %v. Using default backend.", modelName, cudaBackendErr)
		}
		if cudaTargetErr != nil {
			log.Printf("recognition: CUDA Target not available for %s: %v. Using default target.", modelName, cudaTargetErr)
		}

		net.SetPreferableBackend(gocv.NetBackendDefault)
		net.SetPreferableTarget(gocv.NetTargetCPU)
		log.Printf("recognition: Set backend/target to CPU (Default) for %s", modelName)
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
		ScaleFactor: 1.0,
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

// ExtractEmbedding extracts a face embedding from a face region
func (f *FaceRecognitionModel) ExtractEmbedding(faceRegion gocv.Mat) []float32 {
	if f == nil || !f.Enabled || faceRegion.Empty() {
		log.Printf("recognition: ExtractEmbedding called with invalid parameters - f=%v, enabled=%v, faceRegion.Empty()=%v", f != nil, f != nil && f.Enabled, faceRegion.Empty())
		return nil
	}

	log.Printf("recognition: Starting embedding extraction for face region %dx%d", faceRegion.Cols(), faceRegion.Rows())

	// Preprocess face region
	processed := f.preprocessFace(faceRegion)
	if processed.Empty() {
		log.Printf("recognition: ERROR - preprocessFace returned empty matrix")
		return nil
	}
	defer processed.Close()

	log.Printf("recognition: Preprocessed face to %dx%d", processed.Cols(), processed.Rows())

	// Create input blob
	// For ArcFace/FaceNet, we use scale factor to normalize to [0,1] range
	var blob gocv.Mat
	if f.ModelName == "arcface" || f.ModelName == "facenet" {
		blob = gocv.BlobFromImage(processed, 1.0/255.0, image.Pt(f.InputSizeW, f.InputSizeH), gocv.NewScalar(0, 0, 0, 0), false, false)
		log.Printf("recognition: Created blob for %s with scale 1.0/255.0, size %dx%d", f.ModelName, f.InputSizeW, f.InputSizeH)
	} else {
		blob = gocv.BlobFromImage(processed, f.ScaleFactor, image.Pt(f.InputSizeW, f.InputSizeH), f.MeanVal, false, false)
		log.Printf("recognition: Created blob with scale %f, mean %v, size %dx%d", f.ScaleFactor, f.MeanVal, f.InputSizeW, f.InputSizeH)
	}
	defer blob.Close()

	log.Printf("recognition: Created blob with shape %v", blob.Size())

	// Debug: check blob values
	if !blob.Empty() {
		sizes := blob.Size()
		if len(sizes) >= 4 {
			log.Printf("recognition: Blob shape: [%d, %d, %d, %d]", sizes[0], sizes[1], sizes[2], sizes[3])
			// Sample a few values from the blob
			if sizes[0] > 0 && sizes[1] > 0 && sizes[2] > 0 && sizes[3] > 0 {
				// For 4D blob [batch, channel, height, width], we need to calculate the offset
				// Sample first few values
				val1 := blob.GetFloatAt(0, 0)
				val2 := blob.GetFloatAt(0, minInt(56*56, blob.Cols()-1))
				val3 := blob.GetFloatAt(0, minInt(111*111, blob.Cols()-1))
				log.Printf("recognition: Blob sample values: [0,0]=%f, [0,%d]=%f, [0,%d]=%f", val1, minInt(56*56, blob.Cols()-1), val2, minInt(111*111, blob.Cols()-1), val3)
			}
		}
	}

	f.Net.SetInput(blob, "")
	log.Printf("recognition: Set input to network")

	output := f.Net.Forward("")
	defer output.Close()

	log.Printf("recognition: Model output shape: %v", output.Size())

	// Extract embedding vector
	embedding := f.extractEmbeddingVector(output)

	log.Printf("recognition: Extracted embedding vector of length %d", len(embedding))

	// Debug: print first few values and statistics
	if len(embedding) > 0 {
		log.Printf("recognition: First 10 embedding values: %v", embedding[:minInt(10, len(embedding))])

		// Calculate statistics
		var min, max, sum float32
		min = embedding[0]
		max = embedding[0]
		for _, val := range embedding {
			if val < min {
				min = val
			}
			if val > max {
				max = val
			}
			sum += val
		}
		mean := sum / float32(len(embedding))
		log.Printf("recognition: Embedding stats - min: %f, max: %f, mean: %f", min, max, mean)

		// Check if all values are zero
		allZero := true
		for _, val := range embedding {
			if val != 0 {
				allZero = false
				break
			}
		}
		if allZero {
			log.Printf("recognition: WARNING - All embedding values are zero!")
		}
	}

	// Normalize embedding to unit length (L2 normalization)
	if len(embedding) > 0 {
		embedding = f.normalizeEmbedding(embedding)
		log.Printf("recognition: Normalized embedding, first 5 values: %v", embedding[:minInt(5, len(embedding))])
	}

	return embedding
}

// preprocessFace prepares a face region for embedding extraction
func (f *FaceRecognitionModel) preprocessFace(faceRegion gocv.Mat) gocv.Mat {
	if faceRegion.Empty() {
		log.Printf("recognition: ERROR - faceRegion is empty")
		return gocv.Mat{}
	}

	log.Printf("recognition: Preprocessing face region %dx%d, channels: %d", faceRegion.Cols(), faceRegion.Rows(), faceRegion.Channels())

	// Convert BGR to RGB (ArcFace expects RGB input)
	var processed gocv.Mat
	if faceRegion.Channels() == 3 {
		processed = gocv.NewMat()
		gocv.CvtColor(faceRegion, &processed, gocv.ColorBGRToRGB)
		log.Printf("recognition: Converted BGR to RGB")
	} else {
		processed = faceRegion.Clone()
		log.Printf("recognition: Cloned face region (not BGR)")
	}

	// Apply face alignment if landmarks are available
	// For now, we'll just resize the face region
	aligned := gocv.NewMat()
	gocv.Resize(processed, &aligned, image.Pt(f.InputSizeW, f.InputSizeH), 0, 0, gocv.InterpolationLinear)
	log.Printf("recognition: Resized to %dx%d", aligned.Cols(), aligned.Rows())

	// For ArcFace/FaceNet, convert to float32 for better precision
	if f.ModelName == "arcface" || f.ModelName == "facenet" {
		normalized := gocv.NewMat()
		aligned.ConvertTo(&normalized, gocv.MatTypeCV32F)
		aligned.Close()
		aligned = normalized
		log.Printf("recognition: Converted to float32 for %s", f.ModelName)
	}

	// Debug: check pixel values
	if !aligned.Empty() {
		// Sample a few pixel values to verify preprocessing
		rows := aligned.Rows()
		cols := aligned.Cols()
		if rows > 0 && cols > 0 {
			centerRow := rows / 2
			centerCol := cols / 2
			if aligned.Channels() == 3 {
				b := aligned.GetVecbAt(centerRow, centerCol)[0]
				g := aligned.GetVecbAt(centerRow, centerCol)[1]
				r := aligned.GetVecbAt(centerRow, centerCol)[2]
				log.Printf("recognition: Center pixel (BGR): [%d, %d, %d]", b, g, r)
			} else if aligned.Type() == gocv.MatTypeCV32F {
				val := aligned.GetFloatAt(centerRow, centerCol)
				log.Printf("recognition: Center pixel (float32): %f", val)
			}
		}
	}

	processed.Close()
	return aligned
}

// extractEmbeddingVector extracts the embedding vector from model output
func (f *FaceRecognitionModel) extractEmbeddingVector(output gocv.Mat) []float32 {
	sizes := output.Size()
	if len(sizes) == 0 {
		return nil
	}

	// Flatten the output to get the embedding vector
	flattened := output.Reshape(1, 1)
	defer flattened.Close()

	// Extract the embedding values
	embeddingSize := flattened.Cols()
	embedding := make([]float32, embeddingSize)

	for i := 0; i < embeddingSize; i++ {
		embedding[i] = flattened.GetFloatAt(0, i)
	}

	return embedding
}

// normalizeEmbedding normalizes the embedding vector to unit length
func (f *FaceRecognitionModel) normalizeEmbedding(embedding []float32) []float32 {
	if len(embedding) == 0 {
		return embedding
	}

	// Calculate L2 norm
	var norm float32
	for _, val := range embedding {
		norm += val * val
	}
	norm = float32(math.Sqrt(float64(norm)))

	if norm == 0 {
		return embedding
	}

	// Normalize
	normalized := make([]float32, len(embedding))
	for i, val := range embedding {
		normalized[i] = val / norm
	}

	return normalized
}

// CalculateSimilarity calculates cosine similarity between two embeddings
func (f *FaceRecognitionModel) CalculateSimilarity(embedding1, embedding2 []float32) float32 {
	if len(embedding1) != len(embedding2) || len(embedding1) == 0 {
		return 0.0
	}

	var dotProduct float32
	for i := 0; i < len(embedding1); i++ {
		dotProduct += embedding1[i] * embedding2[i]
	}

	// Since embeddings are normalized, dot product equals cosine similarity
	return dotProduct
}

// FindSimilarFaces finds faces similar to a given embedding
func (f *FaceRecognitionModel) FindSimilarFaces(targetEmbedding []float32, candidateEmbeddings [][]float32, threshold float32) []int {
	var similarIndices []int

	for i, candidateEmbedding := range candidateEmbeddings {
		similarity := f.CalculateSimilarity(targetEmbedding, candidateEmbedding)
		if similarity >= threshold {
			similarIndices = append(similarIndices, i)
		}
	}

	return similarIndices
}
