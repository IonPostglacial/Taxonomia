package dataset

import (
	"embed"
	"io/fs"
)

//go:embed static/*
var staticFiles embed.FS

func init() {
	fs.WalkDir(staticFiles, "static", func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			content, err := fs.ReadFile(staticFiles, path)
			if err != nil {
				return err
			}
			staticFileCache.Set("/"+path, content)
		}
		return nil
	})
}
