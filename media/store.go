package media

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// key prefixes for every kind of object kept in the bucket
const (
	PrefixOriginals  = "originals/"
	PrefixThumbnails = "thumbnails/"
	PrefixPreviews   = "previews/"
	PrefixBanners    = "album_banners/"
	PrefixArchives   = "album_archives/"
)

// ErrNotFound is returned when an object does not exist.
var ErrNotFound = errors.New("object not found")

// StoreConfig holds the connection settings for an S3-compatible endpoint.
type StoreConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	Region    string
	UseSSL    bool
}

// Store saves and serves every media asset (originals, thumbnails, previews,
// banners, archives) from a single S3 bucket.
type Store struct {
	client *minio.Client
	bucket string
}

// NewStore connects to the object store and makes sure the bucket exists.
func NewStore(ctx context.Context, cfg StoreConfig) (*Store, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create s3 client: %w", err)
	}

	s := &Store{client: client, bucket: cfg.Bucket}

	var lastErr error
	for attempt := 1; attempt <= 30; attempt++ {
		if lastErr = s.ensureBucket(ctx, cfg.Region); lastErr == nil {
			log.Printf("media.store: using bucket %q at %s", cfg.Bucket, cfg.Endpoint)
			return s, nil
		}
		log.Printf("media.store: bucket check attempt %d failed: %v", attempt, lastErr)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	return nil, fmt.Errorf("object store not reachable: %w", lastErr)
}

func (s *Store) ensureBucket(ctx context.Context, region string) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{Region: region})
}

// Put uploads data under key. size may be -1 when unknown. Returns the stored size.
func (s *Store) Put(ctx context.Context, key string, data io.Reader, size int64, contentType string) (int64, error) {
	info, err := s.client.PutObject(ctx, s.bucket, key, data, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return 0, fmt.Errorf("failed to put object %s: %w", key, err)
	}
	return info.Size, nil
}

// Get opens an object for reading. The returned object supports Seek/ReadAt so it
// can be handed straight to http.ServeContent.
func (s *Store) Get(ctx context.Context, key string) (*minio.Object, minio.ObjectInfo, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, minio.ObjectInfo{}, s.wrapErr(key, err)
	}
	info, err := obj.Stat()
	if err != nil {
		obj.Close()
		return nil, minio.ObjectInfo{}, s.wrapErr(key, err)
	}
	return obj, info, nil
}

// Stat returns object metadata.
func (s *Store) Stat(ctx context.Context, key string) (minio.ObjectInfo, error) {
	info, err := s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return minio.ObjectInfo{}, s.wrapErr(key, err)
	}
	return info, nil
}

// Download copies an object to a temporary file and returns its path along with
// a cleanup func. Used by code that needs a real file (OpenCV, EXIF parsing).
func (s *Store) Download(ctx context.Context, key string) (string, func(), error) {
	obj, _, err := s.Get(ctx, key)
	if err != nil {
		return "", func() {}, err
	}
	defer obj.Close()

	tmp, err := os.CreateTemp("", "mediasys-*"+path.Ext(key))
	if err != nil {
		return "", func() {}, fmt.Errorf("failed to create temp file: %w", err)
	}
	cleanup := func() { os.Remove(tmp.Name()) }
	if _, err := io.Copy(tmp, obj); err != nil {
		tmp.Close()
		cleanup()
		return "", func() {}, fmt.Errorf("failed to download %s: %w", key, err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return "", func() {}, err
	}
	return tmp.Name(), cleanup, nil
}

// Delete removes a single object. Missing objects are not an error.
func (s *Store) Delete(ctx context.Context, key string) error {
	if key == "" {
		return nil
	}
	err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
	if err != nil && !errors.Is(s.wrapErr(key, err), ErrNotFound) {
		return fmt.Errorf("failed to delete object %s: %w", key, err)
	}
	return nil
}

// DeleteKeys removes many objects, logging (not returning) individual failures.
func (s *Store) DeleteKeys(ctx context.Context, keys []string) {
	if len(keys) == 0 {
		return
	}
	ch := make(chan minio.ObjectInfo)
	go func() {
		defer close(ch)
		for _, k := range keys {
			if k != "" {
				ch <- minio.ObjectInfo{Key: k}
			}
		}
	}()
	for res := range s.client.RemoveObjects(ctx, s.bucket, ch, minio.RemoveObjectsOptions{}) {
		log.Printf("media.store: failed to delete %s: %v", res.ObjectName, res.Err)
	}
}

// DeletePrefix removes every object whose key starts with prefix.
func (s *Store) DeletePrefix(ctx context.Context, prefix string) {
	if prefix == "" || !strings.HasSuffix(prefix, "/") {
		log.Printf("media.store: refusing to delete unsafe prefix %q", prefix)
		return
	}
	objects := s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{Prefix: prefix, Recursive: true})
	for res := range s.client.RemoveObjects(ctx, s.bucket, objects, minio.RemoveObjectsOptions{}) {
		log.Printf("media.store: failed to delete %s: %v", res.ObjectName, res.Err)
	}
}

// Ping checks that the bucket is reachable.
func (s *Store) Ping(ctx context.Context) error {
	_, err := s.client.BucketExists(ctx, s.bucket)
	return err
}

func (s *Store) wrapErr(key string, err error) error {
	if minio.ToErrorResponse(err).Code == "NoSuchKey" {
		return fmt.Errorf("%w: %s", ErrNotFound, key)
	}
	return err
}

// OriginalKey returns the object key for an image's original file.
func OriginalKey(imagePath string) string {
	return PrefixOriginals + strings.TrimPrefix(imagePath, "/")
}
