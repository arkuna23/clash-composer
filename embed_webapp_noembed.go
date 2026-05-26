//go:build noembed

package main

import "io/fs"

// webappFS returns nil under the `noembed` build tag, so the binary ships
// without an embedded frontend and only exposes the JSON API at /api/.
func webappFS() fs.FS {
	return nil
}
