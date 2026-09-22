package studio

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var embeddedDist embed.FS

// DistFS returns the embedded Vite build output directory (studio/dist).
func DistFS() (fs.FS, error) {
	return fs.Sub(embeddedDist, "dist")
}
