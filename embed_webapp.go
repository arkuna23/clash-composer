//go:build !noembed

package main

import (
	"embed"
	"io/fs"
)

//go:embed all:webapp/dist
var webappDistFS embed.FS

// webappFS returns the embedded SPA filesystem rooted at webapp/dist, or nil
// if the build output is missing index.html (e.g. the frontend has not been
// built yet — only the placeholder is present).
func webappFS() fs.FS {
	sub, err := fs.Sub(webappDistFS, "webapp/dist")
	if err != nil {
		return nil
	}
	if _, err := fs.Stat(sub, "index.html"); err != nil {
		return nil
	}
	return sub
}
