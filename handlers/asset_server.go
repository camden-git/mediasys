package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/camden-git/mediasysbackend/media"
	"github.com/go-chi/chi/v5"
)

const immutableCacheControl = "public, max-age=31536000, immutable"

// wildcardKey extracts and validates the "*" route segment as an object key suffix.
func wildcardKey(r *http.Request) (string, bool) {
	// chi matches on RawPath when it is set (the param is still escaped) and on
	// the already-decoded Path otherwise, so decode exactly once, and only in
	// the first case.
	decoded := chi.URLParam(r, "*")
	if r.URL.RawPath != "" {
		var err error
		decoded, err = url.PathUnescape(decoded)
		if err != nil {
			return "", false
		}
	}
	clean := strings.TrimPrefix(path.Clean("/"+decoded), "/")
	if clean == "" || strings.HasPrefix(clean, "..") {
		return "", false
	}
	return clean, true
}

// serveObject streams an object from the store with range, ETag and
// conditional request support.
//
// When etag is non-empty it replaces the storage ETag and is checked against
// If-None-Match before the object is opened, so revalidations cost no storage
// round trip.
func serveObject(w http.ResponseWriter, r *http.Request, store *media.Store, key, cacheControl, disposition, etag string) {
	if etag != "" && ifNoneMatch(r, etag) {
		w.Header().Set("ETag", etag)
		if cacheControl != "" {
			w.Header().Set("Cache-Control", cacheControl)
		}
		w.WriteHeader(http.StatusNotModified)
		return
	}
	obj, info, err := store.Get(r.Context(), key)
	if err != nil {
		if errors.Is(err, media.ErrNotFound) {
			WriteAPIError(w, http.StatusNotFound, "ObjectNotFound", "File not found")
			return
		}
		log.Printf("serveObject: failed to open %s: %v", key, err)
		WriteAPIError(w, http.StatusInternalServerError, "ObjectOpenError", "Internal Server Error")
		return
	}
	defer obj.Close()

	if info.ContentType != "" {
		w.Header().Set("Content-Type", info.ContentType)
	}
	if etag != "" {
		w.Header().Set("ETag", etag)
	} else if info.ETag != "" {
		w.Header().Set("ETag", `"`+strings.Trim(info.ETag, `"`)+`"`)
	}
	if cacheControl != "" {
		w.Header().Set("Cache-Control", cacheControl)
	}
	if disposition != "" {
		w.Header().Set("Content-Disposition", disposition)
	}
	http.ServeContent(w, r, path.Base(key), info.LastModified, obj)
}

// ObjectServer serves objects under prefix (e.g. "thumbnails/") from the route
// wildcard. Generated assets have random names, so they are cached forever.
func ObjectServer(store *media.Store, prefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key, ok := wildcardKey(r)
		if !ok {
			WriteAPIError(w, http.StatusBadRequest, "InvalidPath", "Invalid asset path")
			return
		}
		serveObject(w, r, store, prefix+key, immutableCacheControl, "", "")
	}
}

// keyETag derives a strong ETag from an object key. Keys are unique per upload,
// so the tag changes whenever an image is replaced.
func keyETag(key string) string {
	sum := sha256.Sum256([]byte(key))
	return `"` + hex.EncodeToString(sum[:12]) + `"`
}

// ifNoneMatch reports whether the request's If-None-Match header matches etag.
func ifNoneMatch(r *http.Request, etag string) bool {
	header := r.Header.Get("If-None-Match")
	if header == "" {
		return false
	}
	for _, candidate := range strings.Split(header, ",") {
		candidate = strings.TrimPrefix(strings.TrimSpace(candidate), "W/")
		if candidate == "*" || candidate == etag {
			return true
		}
	}
	return false
}

// contentDisposition builds an RFC 6266 header value for filename.
func contentDisposition(kind, filename string) string {
	ascii := strings.Map(func(r rune) rune {
		if r < 0x20 || r > 0x7e || r == '"' || r == '\\' {
			return '_'
		}
		return r
	}, filename)
	return fmt.Sprintf(`%s; filename="%s"; filename*=UTF-8''%s`, kind, ascii, url.PathEscape(filename))
}

// cacheFor returns a public Cache-Control value for d.
func cacheFor(d time.Duration) string {
	return fmt.Sprintf("public, max-age=%d", int(d.Seconds()))
}
