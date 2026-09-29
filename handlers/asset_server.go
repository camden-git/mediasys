package handlers

import (
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
	raw := chi.URLParam(r, "*")
	decoded, err := url.PathUnescape(raw)
	if err != nil {
		return "", false
	}
	clean := strings.TrimPrefix(path.Clean("/"+decoded), "/")
	if clean == "" || strings.HasPrefix(clean, "..") {
		return "", false
	}
	return clean, true
}

// serveObject streams an object from the store with range, ETag and
// conditional request support.
func serveObject(w http.ResponseWriter, r *http.Request, store *media.Store, key, cacheControl, disposition string) {
	obj, info, err := store.Get(r.Context(), key)
	if err != nil {
		if errors.Is(err, media.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		log.Printf("serveObject: failed to open %s: %v", key, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer obj.Close()

	if info.ContentType != "" {
		w.Header().Set("Content-Type", info.ContentType)
	}
	if info.ETag != "" {
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
			http.Error(w, "Invalid asset path", http.StatusBadRequest)
			return
		}
		serveObject(w, r, store, prefix+key, immutableCacheControl, "")
	}
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
