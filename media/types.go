// media/types.go
package media

type AssetType string

const (
	AssetTypeThumbnail AssetType = "thumbnail"
	AssetTypeBanner    AssetType = "banner"
	AssetTypeArchive   AssetType = "archive"
	AssetTypeOriginal  AssetType = "original"
	AssetTypeVideo     AssetType = "video" // soon tm
	AssetTypeUnknown   AssetType = "unknown"
)

// ProcessingOptions can hold parameters for transformations
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

type DetectionResult struct {
	X          int
	Y          int
	W          int
	H          int
	Confidence float32
}
