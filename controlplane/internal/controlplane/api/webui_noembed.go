//go:build !embed_webui
// +build !embed_webui

package api

import (
	"io/fs"
	"os"
)

func init() {
	GetWebUIFS = func() fs.FS {
		// In development, serve from the web-ui/build directory
		return os.DirFS("../web-ui/build")
	}
}
