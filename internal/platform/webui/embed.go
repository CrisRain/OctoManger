package webui

import (
	"embed"
	"io/fs"
	"os"
)

//go:embed all:dist
var embeddedAssets embed.FS

func Open() (fs.FS, string) {
	const localDist = "apps/web/dist"
	if info, err := os.Stat(localDist); err == nil && info.IsDir() {
		return os.DirFS(localDist), "filesystem"
	}

	if assets, ok := openEmbeddedAssets(); ok {
		return assets, "embedded"
	}

	return nil, ""
}

func openEmbeddedAssets() (fs.FS, bool) {
	return openEmbeddedAssetsFrom(embeddedAssets)
}

func openEmbeddedAssetsFrom(source fs.FS) (fs.FS, bool) {
	assets, err := fs.Sub(source, "dist")
	if err != nil {
		return nil, false
	}
	if _, err := fs.Stat(assets, "index.html"); err != nil {
		return nil, false
	}
	return assets, true
}
