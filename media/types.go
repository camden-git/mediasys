package media

type AssetType string

const (
	AssetTypeThumbnail AssetType = "thumbnail"
	AssetTypeBanner    AssetType = "banner"
	AssetTypeArchive   AssetType = "archive"
)

// ImageProcessingOptions holds parameters for transformations
type ImageProcessingOptions struct {
	TargetWidth  int
	TargetHeight int // 0 preserves aspect ratio
	MaxSize      int
	Quality      int
	Format       string // defaults to jpeg
}

// Metadata struct
// Contains EXIF and dimension information
type Metadata struct {
	Width        *int     `json:"width,omitempty"`
	Height       *int     `json:"height,omitempty"`
	Aperture     *float64 `json:"aperture,omitempty"`
	ShutterSpeed *string  `json:"shutter_speed,omitempty"`
	ISO          *int     `json:"iso,omitempty"`
	FocalLength  *float64 `json:"focal_length,omitempty"`
	LensMake     *string  `json:"lens_make,omitempty"`
	LensModel    *string  `json:"lens_model,omitempty"`
	CameraMake   *string  `json:"camera_make,omitempty"`
	CameraModel  *string  `json:"camera_model,omitempty"`
	TakenAt      *int64   `json:"taken_at,omitempty"`
}

// DetectionResult represents a detected face with enhanced information
type DetectionResult struct {
	X          int
	Y          int
	W          int
	H          int
	Confidence float32

	// Enhanced face detection fields
	QualityScore *float32  `json:"quality_score,omitempty"`
	Landmarks    []Point2D `json:"landmarks,omitempty"`  // 5 facial landmarks (eyes, nose, mouth corners)
	PoseYaw      *float32  `json:"pose_yaw,omitempty"`   // Yaw angle in degrees
	PosePitch    *float32  `json:"pose_pitch,omitempty"` // Pitch angle in degrees
	PoseRoll     *float32  `json:"pose_roll,omitempty"`  // Roll angle in degrees

	// Face recognition fields
	Embedding []float32 `json:"embedding,omitempty"`  // 128-dimensional face embedding
	ModelName string    `json:"model_name,omitempty"` // Name of the recognition model used
}

// Point2D represents a 2D point with x,y coordinates
type Point2D struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
}
