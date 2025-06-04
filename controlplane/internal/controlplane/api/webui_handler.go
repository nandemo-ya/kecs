package api

import (
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// WebUIHandler handles serving the Web UI static files and WebSocket connections
type WebUIHandler struct {
	fileSystem fs.FS
	indexPath  string
}

// NewWebUIHandler creates a new Web UI handler
func NewWebUIHandler(fileSystem fs.FS) *WebUIHandler {
	return &WebUIHandler{
		fileSystem: fileSystem,
		indexPath:  "index.html",
	}
}

// ServeHTTP serves the Web UI static files
func (h *WebUIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Clean the path
	upath := r.URL.Path
	if !strings.HasPrefix(upath, "/") {
		upath = "/" + upath
	}
	upath = path.Clean(upath)

	// Remove /ui prefix if present
	if strings.HasPrefix(upath, "/ui") {
		upath = strings.TrimPrefix(upath, "/ui")
		if upath == "" {
			upath = "/"
		}
	}

	// Try to serve the file
	file, err := h.fileSystem.Open(strings.TrimPrefix(upath, "/"))
	if err != nil {
		// If file not found, serve index.html for client-side routing
		h.serveIndex(w, r)
		return
	}
	defer file.Close()

	// Get file info
	stat, err := file.Stat()
	if err != nil {
		h.serveIndex(w, r)
		return
	}

	// If it's a directory, serve index.html
	if stat.IsDir() {
		h.serveIndex(w, r)
		return
	}

	// Set content type based on file extension
	h.setContentType(w, upath)

	// Serve the file
	if seeker, ok := file.(io.ReadSeeker); ok {
		http.ServeContent(w, r, stat.Name(), stat.ModTime(), seeker)
	} else {
		// If not seekable, just copy the content
		io.Copy(w, file)
	}
}

// serveIndex serves the index.html file
func (h *WebUIHandler) serveIndex(w http.ResponseWriter, r *http.Request) {
	index, err := h.fileSystem.Open(h.indexPath)
	if err != nil {
		http.Error(w, "Web UI not available", http.StatusNotFound)
		return
	}
	defer index.Close()

	stat, err := index.Stat()
	if err != nil {
		http.Error(w, "Web UI not available", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if seeker, ok := index.(io.ReadSeeker); ok {
		http.ServeContent(w, r, h.indexPath, stat.ModTime(), seeker)
	} else {
		// If not seekable, just copy the content
		io.Copy(w, index)
	}
}

// setContentType sets the appropriate content type based on file extension
func (h *WebUIHandler) setContentType(w http.ResponseWriter, path string) {
	ext := strings.ToLower(path[strings.LastIndex(path, ".")	:])
	switch ext {
	case ".html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case ".js":
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	case ".css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	case ".json":
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".gif":
		w.Header().Set("Content-Type", "image/gif")
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	case ".ico":
		w.Header().Set("Content-Type", "image/x-icon")
	case ".woff":
		w.Header().Set("Content-Type", "font/woff")
	case ".woff2":
		w.Header().Set("Content-Type", "font/woff2")
	case ".ttf":
		w.Header().Set("Content-Type", "font/ttf")
	case ".eot":
		w.Header().Set("Content-Type", "application/vnd.ms-fontobject")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}
}

// EnableWebUI checks if the Web UI should be enabled
func EnableWebUI() bool {
	// Can be controlled by environment variable
	return true
}

// GetWebUIFS returns the embedded Web UI file system
// This will be defined in a separate file with go:embed
var GetWebUIFS func() fs.FS