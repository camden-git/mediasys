package config

import (
	"errors"
	"fmt"
	"net/url"
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

	// externally visible base URL (e.g. https://photos.example.com), used for
	// absolute links in share pages; when empty the request's host is used
	PublicURL string

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
// minJWTSecretLen is the shortest JWT_SECRET accepted (HS256 wants >= 256 bits).
const minJWTSecretLen = 32

const knownInsecureJWTSecret = "your_super_secret_key_that_should_be_in_config"

func getEnvOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// envErrs collects invalid env values so LoadConfig can report all of them at once.
type envErrs []error

func (e *envErrs) add(err error) { *e = append(*e, err) }

// envInt reads an integer env var, failing (rather than silently defaulting) on
// malformed or out-of-range values.
func (e *envErrs) envInt(envVar string, defaultVal, minVal int) int {
	valStr := os.Getenv(envVar)
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		e.add(fmt.Errorf("%s must be an integer, got %q", envVar, valStr))
		return defaultVal
	}
	if val < minVal {
		e.add(fmt.Errorf("%s must be >= %d, got %d", envVar, minVal, val))
		return defaultVal
	}
	return val
}

func (e *envErrs) envFloat(envVar string, defaultVal, minVal, maxVal float64) float64 {
	valStr := os.Getenv(envVar)
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.ParseFloat(valStr, 64)
	if err != nil {
		e.add(fmt.Errorf("%s must be a number, got %q", envVar, valStr))
		return defaultVal
	}
	if val < minVal || val > maxVal {
		e.add(fmt.Errorf("%s must be between %g and %g, got %g", envVar, minVal, maxVal, val))
		return defaultVal
	}
	return val
}

func (e *envErrs) envBool(envVar string, defaultVal bool) bool {
	valStr := os.Getenv(envVar)
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.ParseBool(valStr)
	if err != nil {
		e.add(fmt.Errorf("%s must be a boolean, got %q", envVar, valStr))
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
	var errs envErrs

	dbURL := getEnvOrDefault("DATABASE_URL", "postgres://mediasys:mediasys@localhost:5432/mediasys?sslmode=disable")

	s3Endpoint := getEnvOrDefault("S3_ENDPOINT", "localhost:9000")
	s3AccessKey := getEnvOrDefault("S3_ACCESS_KEY", "")
	s3SecretKey := getEnvOrDefault("S3_SECRET_KEY", "")
	if s3AccessKey == "" || s3SecretKey == "" {
		return Config{}, fmt.Errorf("S3_ACCESS_KEY and S3_SECRET_KEY must be set")
	}

	databaseDebug := errs.envBool("DATABASE_DEBUG", false)
	s3UseSSL := errs.envBool("S3_USE_SSL", false)
	maxUploadMB := errs.envInt("MAX_UPLOAD_SIZE_MB", 200, 1)

	thumbMaxSize := errs.envInt("THUMBNAIL_MAX_SIZE", defaultThumbnailMaxSize, 1)

	queueSize := errs.envInt("THUMBNAIL_QUEUE_SIZE", defaultThumbnailQueueSize, 1)
	numWorkers := errs.envInt("NUM_THUMBNAIL_WORKERS", defaultNumThumbnailWorkers, 1)
	numDetectionWorkers := errs.envInt("NUM_DETECTION_WORKERS", defaultNumDetectionWorkers, 1)
	detectionQueueSize := errs.envInt("DETECTION_QUEUE_SIZE", defaultDetectionQueueSize, 1)
	memoryTrimInterval := errs.envInt("MEMORY_TRIM_INTERVAL_MINUTES", 30, 0)

	// Legacy DNN face detection
	faceDNNConfig := getEnvOrDefault("FACE_DNN_CONFIG_PATH", "./models/deploy.prototxt")
	faceDNNModel := getEnvOrDefault("FACE_DNN_MODEL_PATH", "./models/res10_300x300_ssd_iter_140000_fp16.caffemodel")

	// New RetinaFace detection
	retinaFaceModel := getEnvOrDefault("RETINAFACE_MODEL_PATH", "./models/retinaface.onnx")

	// Face recognition
	faceRecognitionModel := getEnvOrDefault("FACE_RECOGNITION_MODEL_PATH", "./models/arcface.onnx")
	faceRecognitionModelName := getEnvOrDefault("FACE_RECOGNITION_MODEL_NAME", "arcface")
	faceRecognitionThreshold := errs.envFloat("FACE_RECOGNITION_THRESHOLD", 0.6, 0, 1)
	faceRecognitionEnabled := errs.envBool("FACE_RECOGNITION_ENABLED", true)

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
	if len(jwtSecret) < minJWTSecretLen {
		return Config{}, fmt.Errorf("JWT_SECRET must be at least %d characters (got %d); generate one with: openssl rand -hex 32", minJWTSecretLen, len(jwtSecret))
	}

	if err := errors.Join(errs...); err != nil {
		return Config{}, fmt.Errorf("invalid configuration: %w", err)
	}

	publicURL := strings.TrimRight(strings.TrimSpace(os.Getenv("PUBLIC_URL")), "/")
	if publicURL != "" {
		u, err := url.Parse(publicURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return Config{}, fmt.Errorf("PUBLIC_URL must be an absolute http(s) URL, got %q", publicURL)
		}
	}

	cfg := Config{
		DatabaseURL:               dbURL,
		DatabaseDebug:             databaseDebug,
		S3Endpoint:                s3Endpoint,
		S3AccessKey:               s3AccessKey,
		S3SecretKey:               s3SecretKey,
		S3Bucket:                  getEnvOrDefault("S3_BUCKET", "mediasys"),
		S3Region:                  getEnvOrDefault("S3_REGION", "us-east-1"),
		S3UseSSL:                  s3UseSSL,
		MaxUploadSize:             int64(maxUploadMB) << 20,
		PublicURL:                 publicURL,
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
