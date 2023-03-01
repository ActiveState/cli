package assets

import (
	"embed"
	iofs "io/fs"
)

const (
	// PlaceholderFileName is the name of the file that is used to populate the directory structure of the embedded assets.
	PlaceholderFileName = "placeholder"
)

//go:embed contents/*
var fs embed.FS

type AssetsFS struct {
	fs embed.FS
}

func NewAssetsFS() *AssetsFS {
	return &AssetsFS{fs: fs}
}

func (a *AssetsFS) ReadDir(name string) ([]iofs.DirEntry, error) {
	return a.fs.ReadDir("contents/" + name)
}

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
