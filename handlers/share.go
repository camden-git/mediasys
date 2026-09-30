package handlers

import (
	"net/http"
	"strings"
)

// sharePage describes the minimal Open Graph/Twitter preview page served to link unfurlers.
type sharePage struct {
	Title    string
	Desc     string
	PageURL  string
	ImageURL string // optional
	LinkText string
}

// absoluteURLFunc returns a function that turns a site-relative path into an absolute
// URL. When publicURL is set it is used as the base; otherwise the scheme and host
// the request arrived with are used (honouring X-Forwarded-* headers, which are
// client controlled unless a trusted proxy overwrites them).
func absoluteURLFunc(r *http.Request, publicURL string) func(string) string {
	scheme := "http"
	if r.Header.Get("X-Forwarded-Proto") == "https" || r.TLS != nil {
		scheme = "https"
	}
	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}

	base := scheme + "://" + host
	if publicURL != "" {
		base = strings.TrimRight(publicURL, "/")
	}

	return func(path string) string {
		if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
			return path
		}
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		return base + path
	}
}

// writeShareNotFound answers a share link for a missing resource with a small HTML 404,
// since browsers and link unfurlers (not API clients) request these routes.
func writeShareNotFound(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte("<!doctype html><html lang=\"en\"><head><meta charset=\"utf-8\"><title>Not found</title></head>" +
		"<body><p>This link is no longer available.</p></body></html>"))
}

// writeSharePage writes the share page HTML for link unfurlers.
func writeSharePage(w http.ResponseWriter, p sharePage) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	html := "<!doctype html><html lang=\"en\"><head>" +
		"<meta charset=\"utf-8\">" +
		"<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">" +
		"<title>" + htmlEscape(p.Title) + "</title>" +
		"<meta property=\"og:type\" content=\"website\">" +
		"<meta property=\"og:url\" content=\"" + htmlAttr(p.PageURL) + "\">" +
		"<meta property=\"og:title\" content=\"" + htmlAttr(p.Title) + "\">" +
		"<meta property=\"og:description\" content=\"" + htmlAttr(p.Desc) + "\">"
	if p.ImageURL != "" {
		html += "<meta property=\"og:image\" content=\"" + htmlAttr(p.ImageURL) + "\">" +
			"<meta property=\"og:image:alt\" content=\"" + htmlAttr(p.Title) + "\">"
	}
	html += "<meta name=\"twitter:card\" content=\"summary_large_image\">" +
		"<meta name=\"twitter:title\" content=\"" + htmlAttr(p.Title) + "\">" +
		"<meta name=\"twitter:description\" content=\"" + htmlAttr(p.Desc) + "\">"
	if p.ImageURL != "" {
		html += "<meta name=\"twitter:image\" content=\"" + htmlAttr(p.ImageURL) + "\">"
	}
	html += "<meta http-equiv=\"refresh\" content=\"0;url=" + htmlAttr(p.PageURL) + "\">" +
		"</head><body><a href=\"" + htmlAttr(p.PageURL) + "\">" + p.LinkText + "</a></body></html>"

	_, _ = w.Write([]byte(html))
}

// htmlEscape escapes text node content.
func htmlEscape(s string) string {
	return strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
	).Replace(s)
}

// htmlAttr escapes attribute values.
func htmlAttr(s string) string {
	return strings.NewReplacer(
		"&", "&amp;",
		"\"", "&quot;",
		"<", "&lt;",
		">", "&gt;",
	).Replace(s)
}
