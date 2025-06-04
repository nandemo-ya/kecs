//go:build embed_webui
// +build embed_webui

package api

import (
	"embed"
	"io/fs"
)

//go:embed all:webui_dist
var webUIFiles embed.FS

func init() {
	GetWebUIFS = func() fs.FS {
		sub, _ := fs.Sub(webUIFiles, "webui_dist")
		return sub
	}
}