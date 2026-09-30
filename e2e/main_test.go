// Package e2e_test contains end-to-end tests that exercise the real HTTP
// router (handlers.RegisterRoutes, via app.New) against a real Postgres
// server and a real S3-compatible object store.
//
// These tests are skipped unless the following environment variables are
// set:
//
//	TEST_DATABASE_URL  admin connection string, e.g.
//	                    postgres://mediasys:mediasys@localhost:5432/postgres?sslmode=disable
//	                    (must be able to CREATE/DROP DATABASE and CREATE EXTENSION vector;
//	                    see database/migrate_schema_test.go for the same convention)
//	TEST_S3_ENDPOINT    e.g. localhost:9000
//	TEST_S3_ACCESS_KEY
//	TEST_S3_SECRET_KEY
//
// Optional:
//
//	TEST_S3_REGION      defaults to us-east-1
//	TEST_S3_USE_SSL     defaults to false
//
// A quick way to get throwaway Postgres + MinIO-compatible containers locally:
//
//	docker run -d --name mediasys-test-pg -e POSTGRES_USER=mediasys \
//	  -e POSTGRES_PASSWORD=mediasys -e POSTGRES_DB=mediasys -p 15432:5432 pgvector/pgvector:pg17
//	docker run -d --name mediasys-test-minio -e MINIO_ROOT_USER=mediasys \
//	  -e MINIO_ROOT_PASSWORD=change-me-please -p 19000:9000 pgsty/silo:RELEASE.2026-09-16T00-00-00Z \
//	  server /data --console-address ":9001"
//
//	TEST_DATABASE_URL="postgres://mediasys:mediasys@localhost:15432/postgres?sslmode=disable" \
//	TEST_S3_ENDPOINT="localhost:19000" TEST_S3_ACCESS_KEY=mediasys TEST_S3_SECRET_KEY=change-me-please \
//	go test ./e2e/...
//
// Every run of this package creates one fresh scratch database and one fresh
// bucket (random suffix), used by every test in the package, and drops/empties
// them again when the run finishes.
package e2e_test

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/camden-git/mediasysbackend/app"
	"github.com/camden-git/mediasysbackend/config"
)

// sharedEnv is built once per test binary run by TestMain, and used by every
// test in the package.
type sharedEnvT struct {
	server *httptest.Server
	app    *app.App

	adminUsername string
	adminPassword string
	adminToken    string
}

var shared *sharedEnvT

func envReady() bool {
	return os.Getenv("TEST_DATABASE_URL") != "" &&
		os.Getenv("TEST_S3_ENDPOINT") != "" &&
		os.Getenv("TEST_S3_ACCESS_KEY") != "" &&
		os.Getenv("TEST_S3_SECRET_KEY") != ""
}

func TestMain(m *testing.M) {
	code := 1
	defer func() { os.Exit(code) }()

	if envReady() {
		cleanup, err := setupShared()
		if err != nil {
			fmt.Fprintln(os.Stderr, "e2e setup failed:", err)
			return
		}
		defer cleanup()
	}

	code = m.Run()
}

func randomSuffix() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func setupShared() (func(), error) {
	ctx := context.Background()
	adminDBURL := os.Getenv("TEST_DATABASE_URL")

	adminDB, err := sql.Open("pgx", adminDBURL)
	if err != nil {
		return nil, fmt.Errorf("open admin db: %w", err)
	}
	if err := adminDB.PingContext(ctx); err != nil {
		_ = adminDB.Close()
		return nil, fmt.Errorf("ping admin db: %w", err)
	}

	suffix := randomSuffix()
	dbName := "mediasys_e2e_" + suffix
	if _, err := adminDB.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %q", dbName)); err != nil {
		_ = adminDB.Close()
		return nil, fmt.Errorf("create scratch database: %w", err)
	}

	scratchURL, err := url.Parse(adminDBURL)
	if err != nil {
		return nil, err
	}
	scratchURL.Path = "/" + dbName

	bucket := "mediasys-e2e-" + suffix

	// build config.Config the same way LoadConfig would, by populating the
	// same env vars it reads and delegating to it, so any future default
	// changes stay in sync automatically.
	envVars := map[string]string{
		"DATABASE_URL":             scratchURL.String(),
		"S3_ENDPOINT":              os.Getenv("TEST_S3_ENDPOINT"),
		"S3_ACCESS_KEY":            os.Getenv("TEST_S3_ACCESS_KEY"),
		"S3_SECRET_KEY":            os.Getenv("TEST_S3_SECRET_KEY"),
		"S3_BUCKET":                bucket,
		"S3_REGION":                envOrDefault("TEST_S3_REGION", "us-east-1"),
		"S3_USE_SSL":               envOrDefault("TEST_S3_USE_SSL", "false"),
		"JWT_SECRET":               randomSuffix() + randomSuffix() + randomSuffix() + randomSuffix(),
		"FACE_RECOGNITION_ENABLED": "false", // no ONNX models available in the test environment
		"CORS_ALLOWED_ORIGINS":     "http://example.test",
	}
	var unset []string
	for k, v := range envVars {
		if _, had := os.LookupEnv(k); !had {
			unset = append(unset, k)
		}
		_ = os.Setenv(k, v)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	a, err := app.New(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("build app: %w", err)
	}

	server := httptest.NewServer(a.Router)

	env := &sharedEnvT{
		server:        server,
		app:           a,
		adminUsername: "admin_" + suffix,
		adminPassword: "test-password-" + suffix,
	}

	if err := createInitialAdmin(server.URL, env.adminUsername, env.adminPassword); err != nil {
		server.Close()
		_ = a.Close()
		return nil, fmt.Errorf("create initial admin: %w", err)
	}
	token, err := login(server.URL, env.adminUsername, env.adminPassword)
	if err != nil {
		server.Close()
		_ = a.Close()
		return nil, fmt.Errorf("login as initial admin: %w", err)
	}
	env.adminToken = token
	shared = env

	cleanup := func() {
		server.Close()
		_ = a.Close()

		emptyAndDeleteBucket(ctx, bucket)

		if _, err := adminDB.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %q WITH (FORCE)", dbName)); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to drop scratch database %s: %v\n", dbName, err)
		}
		_ = adminDB.Close()

		for _, k := range unset {
			_ = os.Unsetenv(k)
		}
	}
	return cleanup, nil
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func emptyAndDeleteBucket(ctx context.Context, bucket string) {
	client, err := minio.New(os.Getenv("TEST_S3_ENDPOINT"), &minio.Options{
		Creds:  credentials.NewStaticV4(os.Getenv("TEST_S3_ACCESS_KEY"), os.Getenv("TEST_S3_SECRET_KEY"), ""),
		Secure: envOrDefault("TEST_S3_USE_SSL", "false") == "true",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to build S3 client for cleanup: %v\n", err)
		return
	}
	objectsCh := client.ListObjects(ctx, bucket, minio.ListObjectsOptions{Recursive: true})
	for obj := range objectsCh {
		if obj.Err != nil {
			continue
		}
		_ = client.RemoveObject(ctx, bucket, obj.Key, minio.RemoveObjectOptions{})
	}
	if err := client.RemoveBucket(ctx, bucket); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to remove bucket %s: %v\n", bucket, err)
	}
}

// requireShared skips (fails when CI=true) the test if the environment isn't configured for e2e
// tests, and otherwise returns the shared server/app built by TestMain.
func requireShared(t *testing.T) *sharedEnvT {
	t.Helper()
	if !envReady() {
		if os.Getenv("CI") == "true" {
			t.Fatal("TEST_DATABASE_URL and TEST_S3_* env vars are required when CI=true")
		}
		t.Skip("TEST_DATABASE_URL and TEST_S3_* env vars not set; skipping e2e tests (see e2e/main_test.go doc comment)")
	}
	if shared == nil {
		t.Fatal("e2e environment was not set up (see TestMain output for the setup error)")
	}
	return shared
}

// processingTimeout is how long tests wait for background image processing
// (thumbnails, metadata). Override with E2E_PROCESSING_TIMEOUT (a Go duration,
// e.g. 90s); the default is 30s locally and 2m when CI=true.
func processingTimeout() time.Duration {
	if v := os.Getenv("E2E_PROCESSING_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	if os.Getenv("CI") == "true" {
		return 2 * time.Minute
	}
	return 30 * time.Second
}

// pollUntil polls fn every interval until it returns true, or fails the test
// once timeout elapses.
func pollUntil(t *testing.T, timeout, interval time.Duration, msg string, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		if fn() {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for: %s", msg)
		}
		time.Sleep(interval)
	}
}
