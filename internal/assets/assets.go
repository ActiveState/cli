package assets

import (
	"embed"
	iofs "io/fs"
)

//go:embed contents/*
var fs embed.FS

// ReadFileBytes reads and returns bytes from the given file in this package's embedded assets.
// Filenames should use forward slashes, not `filepath.Join()`, because go:embed requires '/'.
func ReadFileBytes(filename string) ([]byte, error) {
	return fs.ReadFile("contents/" + filename)
}

func OpenFile(filename string) (iofs.File, error) {
	return fs.Open("contents/" + filename)
}

func ReadDir(dirname string) ([]iofs.DirEntry, error) {
	return fs.ReadDir("contents/" + dirname)
}
