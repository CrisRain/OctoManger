package apiserver

import (
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
)

const (
	CacheControlNoCache         = "no-cache"
	CacheControlImmutableAssets = "public, max-age=31536000, immutable"
)

type StaticFile struct {
	Path            string
	Body            []byte
	ContentType     string
	ContentEncoding string
	CacheControl    string
}

func ResolveStaticFile(assets fs.FS, method, requestPath, acceptHeader, acceptEncoding string) (StaticFile, bool) {
	relativePath := normalizeStaticRequestPath(requestPath)
	if strings.HasPrefix(relativePath, "api/") {
		return StaticFile{}, false
	}

	if file, ok := resolveExactStaticFile(assets, relativePath, acceptEncoding); ok {
		return file, true
	}

	if !shouldServeSPAFallback(method, relativePath, acceptHeader) {
		return StaticFile{}, false
	}

	return resolveExactStaticFile(assets, "index.html", acceptEncoding)
}

func normalizeStaticRequestPath(requestPath string) string {
	cleanedPath := path.Clean("/" + strings.TrimSpace(requestPath))
	relativePath := strings.TrimPrefix(cleanedPath, "/")
	if relativePath == "" || relativePath == "." {
		return "index.html"
	}
	return relativePath
}

func resolveExactStaticFile(assets fs.FS, relativePath, acceptEncoding string) (StaticFile, bool) {
	if !hasRegularFile(assets, relativePath) {
		return StaticFile{}, false
	}

	selectedPath, contentEncoding := selectCompressedStaticFile(assets, relativePath, acceptEncoding)
	body, err := fs.ReadFile(assets, selectedPath)
	if err != nil {
		return StaticFile{}, false
	}

	return StaticFile{
		Path:            selectedPath,
		Body:            body,
		ContentType:     mime.TypeByExtension(path.Ext(relativePath)),
		ContentEncoding: contentEncoding,
		CacheControl:    cacheControlForStaticPath(relativePath),
	}, true
}

func shouldServeSPAFallback(method, relativePath, acceptHeader string) bool {
	if method != http.MethodGet && method != http.MethodHead {
		return false
	}
	if path.Ext(relativePath) != "" {
		return false
	}

	acceptHeader = strings.ToLower(strings.TrimSpace(acceptHeader))
	return acceptHeader == "" || strings.Contains(acceptHeader, "text/html")
}

func selectCompressedStaticFile(assets fs.FS, relativePath, acceptEncoding string) (string, string) {
	if AcceptsEncoding(acceptEncoding, "br") && hasRegularFile(assets, relativePath+".br") {
		return relativePath + ".br", "br"
	}
	if AcceptsEncoding(acceptEncoding, "gzip") && hasRegularFile(assets, relativePath+".gz") {
		return relativePath + ".gz", "gzip"
	}
	return relativePath, ""
}

func AcceptsEncoding(acceptEncoding, wanted string) bool {
	for _, entry := range strings.Split(acceptEncoding, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		name := entry
		params := ""
		if idx := strings.Index(entry, ";"); idx >= 0 {
			name = strings.TrimSpace(entry[:idx])
			params = entry[idx+1:]
		}
		if !strings.EqualFold(name, wanted) && name != "*" {
			continue
		}
		if !encodingEnabled(params) {
			continue
		}
		return true
	}

	return false
}

func encodingEnabled(params string) bool {
	for _, param := range strings.Split(params, ";") {
		key, value, ok := strings.Cut(strings.TrimSpace(param), "=")
		if !ok || !strings.EqualFold(key, "q") {
			continue
		}

		qValue, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err != nil {
			return true
		}
		return qValue > 0
	}

	return true
}

func cacheControlForStaticPath(relativePath string) string {
	if relativePath == "index.html" {
		return CacheControlNoCache
	}
	if strings.HasPrefix(relativePath, "assets/") {
		return CacheControlImmutableAssets
	}
	return CacheControlNoCache
}

func hasRegularFile(assets fs.FS, filename string) bool {
	info, err := fs.Stat(assets, filename)
	return err == nil && !info.IsDir()
}

func ApplyStaticFileHeaders(c *app.RequestContext, file StaticFile) {
	c.Header("Cache-Control", file.CacheControl)
	c.Header("Vary", "Accept-Encoding")
	c.Header("X-Content-Type-Options", "nosniff")
	if file.ContentType != "" {
		c.Header("Content-Type", file.ContentType)
	}
	if file.ContentEncoding != "" {
		c.Header("Content-Encoding", file.ContentEncoding)
	}
}
