package config

import (
	"net/netip"
	"reflect"
	"strings"
	"testing"
)

const testJWTSecret = "0123456789abcdef0123456789abcdef"

// setValidEnv sets the minimum environment LoadConfig needs to succeed and
// clears optional variables that could leak in from the host environment.
func setValidEnv(t *testing.T) {
	t.Helper()
	t.Setenv("JWT_SECRET", testJWTSecret)
	t.Setenv("S3_ACCESS_KEY", "access")
	t.Setenv("S3_SECRET_KEY", "secret")
	for _, k := range []string{
		"DATABASE_DEBUG", "S3_USE_SSL", "MAX_UPLOAD_SIZE_MB", "THUMBNAIL_MAX_SIZE",
		"THUMBNAIL_QUEUE_SIZE", "NUM_THUMBNAIL_WORKERS", "NUM_DETECTION_WORKERS",
		"DETECTION_QUEUE_SIZE", "MEMORY_TRIM_INTERVAL_MINUTES",
		"FACE_RECOGNITION_THRESHOLD", "FACE_RECOGNITION_ENABLED",
		"PUBLIC_URL", "CORS_ALLOWED_ORIGINS", "TRUSTED_PROXIES",
	} {
		t.Setenv(k, "")
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	setValidEnv(t)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.JWTSecret != testJWTSecret || cfg.S3AccessKey != "access" || cfg.S3SecretKey != "secret" {
		t.Fatalf("required values not loaded: %+v", cfg)
	}
	if cfg.MaxUploadSize != 200<<20 {
		t.Fatalf("MaxUploadSize = %d, want %d", cfg.MaxUploadSize, int64(200)<<20)
	}
	if !cfg.FaceRecognitionEnabled {
		t.Fatalf("FaceRecognitionEnabled should default to true")
	}
	if cfg.PublicURL != "" {
		t.Fatalf("PublicURL = %q, want empty", cfg.PublicURL)
	}
}

func TestLoadConfigRequiredSecrets(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantErr string
	}{
		{"missing JWT_SECRET", map[string]string{"JWT_SECRET": ""}, "JWT_SECRET must be set"},
		{"known insecure JWT_SECRET", map[string]string{"JWT_SECRET": knownInsecureJWTSecret}, "known default"},
		{"short JWT_SECRET", map[string]string{"JWT_SECRET": strings.Repeat("a", minJWTSecretLen-1)}, "at least"},
		{"missing S3_ACCESS_KEY", map[string]string{"S3_ACCESS_KEY": ""}, "S3_ACCESS_KEY and S3_SECRET_KEY"},
		{"missing S3_SECRET_KEY", map[string]string{"S3_SECRET_KEY": ""}, "S3_ACCESS_KEY and S3_SECRET_KEY"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setValidEnv(t)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			_, err := LoadConfig()
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestLoadConfigJWTSecretAtMinimumLength(t *testing.T) {
	setValidEnv(t)
	t.Setenv("JWT_SECRET", strings.Repeat("a", minJWTSecretLen))
	if _, err := LoadConfig(); err != nil {
		t.Fatalf("secret of exactly %d chars should be accepted: %v", minJWTSecretLen, err)
	}
}

func TestLoadConfigInvalidValues(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		value   string
		wantErr string
	}{
		{"non-integer", "MAX_UPLOAD_SIZE_MB", "abc", "MAX_UPLOAD_SIZE_MB must be an integer"},
		{"integer below minimum", "NUM_THUMBNAIL_WORKERS", "0", "NUM_THUMBNAIL_WORKERS must be >= 1"},
		{"negative trim interval", "MEMORY_TRIM_INTERVAL_MINUTES", "-1", "MEMORY_TRIM_INTERVAL_MINUTES must be >= 0"},
		{"non-number float", "FACE_RECOGNITION_THRESHOLD", "high", "FACE_RECOGNITION_THRESHOLD must be a number"},
		{"float above range", "FACE_RECOGNITION_THRESHOLD", "1.5", "FACE_RECOGNITION_THRESHOLD must be between"},
		{"float below range", "FACE_RECOGNITION_THRESHOLD", "-0.1", "FACE_RECOGNITION_THRESHOLD must be between"},
		{"non-boolean", "S3_USE_SSL", "maybe", "S3_USE_SSL must be a boolean"},
		{"invalid trusted proxy", "TRUSTED_PROXIES", "10.0.0.0/8,proxy.local", "TRUSTED_PROXIES entries must be IPs or CIDRs"},
		{"trusted proxy prefix out of range", "TRUSTED_PROXIES", "10.0.0.0/33", "TRUSTED_PROXIES entries must be IPs or CIDRs"},
		{"empty trusted proxy list", "TRUSTED_PROXIES", " , ", "TRUSTED_PROXIES must list at least one"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setValidEnv(t)
			t.Setenv(tt.key, tt.value)
			_, err := LoadConfig()
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestLoadConfigReportsAllInvalidValues(t *testing.T) {
	setValidEnv(t)
	t.Setenv("MAX_UPLOAD_SIZE_MB", "abc")
	t.Setenv("DATABASE_DEBUG", "nope")
	t.Setenv("FACE_RECOGNITION_THRESHOLD", "2")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{"MAX_UPLOAD_SIZE_MB", "DATABASE_DEBUG", "FACE_RECOGNITION_THRESHOLD"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %s", err, want)
		}
	}
}

func TestLoadConfigValidValues(t *testing.T) {
	setValidEnv(t)
	t.Setenv("MAX_UPLOAD_SIZE_MB", "5")
	t.Setenv("S3_USE_SSL", "true")
	t.Setenv("FACE_RECOGNITION_ENABLED", "false")
	t.Setenv("FACE_RECOGNITION_THRESHOLD", "0.75")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.MaxUploadSize != 5<<20 {
		t.Errorf("MaxUploadSize = %d, want %d", cfg.MaxUploadSize, 5<<20)
	}
	if !cfg.S3UseSSL {
		t.Error("S3UseSSL should be true")
	}
	if cfg.FaceRecognitionEnabled {
		t.Error("FaceRecognitionEnabled should be false")
	}
	if cfg.FaceRecognitionThreshold != 0.75 {
		t.Errorf("FaceRecognitionThreshold = %v, want 0.75", cfg.FaceRecognitionThreshold)
	}
}

func TestLoadConfigPublicURL(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{"unset", "", "", false},
		{"https", "https://media.example.com", "https://media.example.com", false},
		{"trailing slash trimmed", "http://localhost:8080/", "http://localhost:8080", false},
		{"surrounding whitespace trimmed", "  https://media.example.com/  ", "https://media.example.com", false},
		{"missing scheme", "media.example.com", "", true},
		{"unsupported scheme", "ftp://media.example.com", "", true},
		{"missing host", "https://", "", true},
		{"relative path", "/media", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setValidEnv(t)
			t.Setenv("PUBLIC_URL", tt.value)
			cfg, err := LoadConfig()
			if tt.wantErr {
				if err == nil || !strings.Contains(err.Error(), "PUBLIC_URL") {
					t.Fatalf("err = %v, want PUBLIC_URL error", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("LoadConfig: %v", err)
			}
			if cfg.PublicURL != tt.want {
				t.Fatalf("PublicURL = %q, want %q", cfg.PublicURL, tt.want)
			}
		})
	}
}

func TestLoadConfigCORSOrigins(t *testing.T) {
	setValidEnv(t)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if len(cfg.CORSAllowedOrigins) == 0 {
		t.Fatal("expected default CORS origins")
	}

	t.Setenv("CORS_ALLOWED_ORIGINS", " https://a.example.com , ,https://b.example.com,")
	cfg, err = LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	want := []string{"https://a.example.com", "https://b.example.com"}
	if !reflect.DeepEqual(cfg.CORSAllowedOrigins, want) {
		t.Fatalf("CORSAllowedOrigins = %v, want %v", cfg.CORSAllowedOrigins, want)
	}
}

func TestLoadConfigTrustedProxies(t *testing.T) {
	setValidEnv(t)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	for _, ip := range []string{"127.0.0.1", "::1", "10.1.2.3", "172.18.0.2", "192.168.1.1", "fd00::1"} {
		if !containsAddr(cfg.TrustedProxies, ip) {
			t.Errorf("default TrustedProxies should contain %s", ip)
		}
	}
	for _, ip := range []string{"203.0.113.7", "172.32.0.1", "2001:db8::1"} {
		if containsAddr(cfg.TrustedProxies, ip) {
			t.Errorf("default TrustedProxies should not contain %s", ip)
		}
	}

	t.Setenv("TRUSTED_PROXIES", " 173.245.48.0/20 , 10.0.0.5,::ffff:192.0.2.1, 2400:cb00::/32,10.1.2.3/8")
	cfg, err = LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	want := []netip.Prefix{
		netip.MustParsePrefix("173.245.48.0/20"),
		netip.MustParsePrefix("10.0.0.5/32"),
		netip.MustParsePrefix("192.0.2.1/32"),
		netip.MustParsePrefix("2400:cb00::/32"),
		netip.MustParsePrefix("10.0.0.0/8"),
	}
	if !reflect.DeepEqual(cfg.TrustedProxies, want) {
		t.Fatalf("TrustedProxies = %v, want %v", cfg.TrustedProxies, want)
	}
}

func containsAddr(prefixes []netip.Prefix, ip string) bool {
	addr := netip.MustParseAddr(ip)
	for _, p := range prefixes {
		if p.Contains(addr) {
			return true
		}
	}
	return false
}
