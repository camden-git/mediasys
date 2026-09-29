package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

const (
	defaultThumbnailQueueSize  = 50
	defaultNumThumbnailWorkers = 2
	defaultThumbnailMaxSize    = 300
	defaultNumDetectionWorkers = 1
	defaultDetectionQueueSize  = 10
)

type Config struct {
	// postgres connection string, e.g. postgres://user:pass@host:5432/db?sslmode=disable
	DatabaseURL string
	// log every SQL statement when true
	DatabaseDebug bool

	// S3-compatible object storage
	S3Endpoint  string
	S3AccessKey string
	S3SecretKey string
	S3Bucket    string
	S3Region    string
	S3UseSSL    bool

	// maximum size in bytes of a single uploaded original
	MaxUploadSize int64

	// origins allowed to make credentialed requests to auth/admin routes
	CORSAllowedOrigins []string

	// thumbnail generation settings
	ThumbnailMaxSize int

	// worker settings
	ThumbnailQueueSize        int
	NumThumbnailWorkers       int
	NumDetectionWorkers       int
	DetectionQueueSize        int
	MemoryTrimIntervalMinutes int

	// face detection model paths (DNN - legacy)
	FaceDNNNetConfigPath string
	FaceDNNNetModelPath  string

	// face detection model paths (RetinaFace)
	RetinaFaceModelPath string

	// face recognition model paths
	FaceRecognitionModelPath string
	FaceRecognitionModelName string // "arcface", "facenet", etc.

	// face recognition settings
	FaceRecognitionThreshold float64 // similarity threshold for face matching
	FaceRecognitionEnabled   bool    // whether to enable face recognition

	// Cloudflare Turnstile
	TurnstileSiteKey   string
	TurnstileSecretKey string

	// JWT signing secret used to sign and verify auth tokens
	JWTSecret string
}

// knownInsecureJWTSecret is the old hardcoded fallback secret that used to live in
// handlers/auth.go. It must never be accepted as a real secret since it is public
// (visible in source history), so LoadConfig rejects it explicitly.
const knownInsecureJWTSecret = "your_super_secret_key_that_should_be_in_config"

func getEnvOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvIntOrDefault(envVar string, defaultVal int) int {
	valStr := os.Getenv(envVar)
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(valStr)
	if err != nil || val <= 0 {
		log.Printf("Warning: Invalid %s '%s'. Using default %d. Error: %v", envVar, valStr, defaultVal, err)
		return defaultVal
	}
	return val
}

func getEnvFloatOrDefault(envVar string, defaultVal float64) float64 {
	valStr := os.Getenv(envVar)
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.ParseFloat(valStr, 64)
	if err != nil {
		log.Printf("Warning: Invalid %s '%s'. Using default %f. Error: %v", envVar, valStr, defaultVal, err)
		return defaultVal
	}
	return val
}

func getEnvBoolOrDefault(envVar string, defaultVal bool) bool {
	valStr := os.Getenv(envVar)
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.ParseBool(valStr)
	if err != nil {
		log.Printf("Warning: Invalid %s '%s'. Using default %t. Error: %v", envVar, valStr, defaultVal, err)
		return defaultVal
	}
	return val
}

func splitList(v string) []string {
	var out []string
	for _, p := range strings.Split(v, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func LoadConfig() (Config, error) {
	dbURL := getEnvOrDefault("DATABASE_URL", "postgres://mediasys:mediasys@localhost:5432/mediasys?sslmode=disable")

	s3Endpoint := getEnvOrDefault("S3_ENDPOINT", "localhost:9000")
	s3AccessKey := getEnvOrDefault("S3_ACCESS_KEY", "")
	s3SecretKey := getEnvOrDefault("S3_SECRET_KEY", "")
	if s3AccessKey == "" || s3SecretKey == "" {
		return Config{}, fmt.Errorf("S3_ACCESS_KEY and S3_SECRET_KEY must be set")
	}

	thumbMaxSize := getEnvIntOrDefault("THUMBNAIL_MAX_SIZE", defaultThumbnailMaxSize)

	queueSize := getEnvIntOrDefault("THUMBNAIL_QUEUE_SIZE", defaultThumbnailQueueSize)
	numWorkers := getEnvIntOrDefault("NUM_THUMBNAIL_WORKERS", defaultNumThumbnailWorkers)
	numDetectionWorkers := getEnvIntOrDefault("NUM_DETECTION_WORKERS", defaultNumDetectionWorkers)
	detectionQueueSize := getEnvIntOrDefault("DETECTION_QUEUE_SIZE", defaultDetectionQueueSize)
	memoryTrimInterval := getEnvIntOrDefault("MEMORY_TRIM_INTERVAL_MINUTES", 30)

	// Legacy DNN face detection
	faceDNNConfig := getEnvOrDefault("FACE_DNN_CONFIG_PATH", "./models/deploy.prototxt.txt")
	faceDNNModel := getEnvOrDefault("FACE_DNN_MODEL_PATH", "./models/res10_300x300_ssd_iter_140000_fp16.caffemodel")

	// New RetinaFace detection
	retinaFaceModel := getEnvOrDefault("RETINAFACE_MODEL_PATH", "./models/retinaface.onnx")

	// Face recognition
	faceRecognitionModel := getEnvOrDefault("FACE_RECOGNITION_MODEL_PATH", "./models/arcface.onnx")
	faceRecognitionModelName := getEnvOrDefault("FACE_RECOGNITION_MODEL_NAME", "arcface")
	faceRecognitionThreshold := getEnvFloatOrDefault("FACE_RECOGNITION_THRESHOLD", 0.6)
	faceRecognitionEnabled := getEnvBoolOrDefault("FACE_RECOGNITION_ENABLED", true)
	// log.Printf("Config: FACE_RECOGNITION_ENABLED env var parsed as: %v", faceRecognitionEnabled)

	// Cloudflare Turnstile
	turnstileSiteKey := getEnvOrDefault("TURNSTILE_SITE_KEY", "")
	turnstileSecretKey := getEnvOrDefault("TURNSTILE_SECRET_KEY", "")

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return Config{}, fmt.Errorf("JWT_SECRET must be set to a strong, random value")
	}
	if jwtSecret == knownInsecureJWTSecret {
		return Config{}, fmt.Errorf("JWT_SECRET must not be set to the known default value; generate a new secret")
	}

	cfg := Config{
		DatabaseURL:               dbURL,
		DatabaseDebug:             getEnvBoolOrDefault("DATABASE_DEBUG", false),
		S3Endpoint:                s3Endpoint,
		S3AccessKey:               s3AccessKey,
		S3SecretKey:               s3SecretKey,
		S3Bucket:                  getEnvOrDefault("S3_BUCKET", "mediasys"),
		S3Region:                  getEnvOrDefault("S3_REGION", "us-east-1"),
		S3UseSSL:                  getEnvBoolOrDefault("S3_USE_SSL", false),
		MaxUploadSize:             int64(getEnvIntOrDefault("MAX_UPLOAD_SIZE_MB", 200)) << 20,
		CORSAllowedOrigins:        splitList(getEnvOrDefault("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173,http://localhost:4173,http://127.0.0.1:4173")),
		ThumbnailMaxSize:          thumbMaxSize,
		ThumbnailQueueSize:        queueSize,
		NumThumbnailWorkers:       numWorkers,
		NumDetectionWorkers:       numDetectionWorkers,
		DetectionQueueSize:        detectionQueueSize,
		MemoryTrimIntervalMinutes: memoryTrimInterval,
		FaceDNNNetConfigPath:      faceDNNConfig,
		FaceDNNNetModelPath:       faceDNNModel,
		RetinaFaceModelPath:       retinaFaceModel,
		FaceRecognitionModelPath:  faceRecognitionModel,
		FaceRecognitionModelName:  faceRecognitionModelName,
		FaceRecognitionThreshold:  faceRecognitionThreshold,
		FaceRecognitionEnabled:    faceRecognitionEnabled,
		TurnstileSiteKey:          turnstileSiteKey,
		TurnstileSecretKey:        turnstileSecretKey,
		JWTSecret:                 jwtSecret,
	}

	return cfg, nil
}
