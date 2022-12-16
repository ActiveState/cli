package unarchiver

import (
	"archive/zip"
	"fmt"
	"path/filepath"

	"github.com/ActiveState/archiver"
)

/*
  This file implements an extension of the `archiver.Zip` type that unarchives a zip archive
  reporting its progress.
*/

// ensure that it implements the SingleUnarchiver interface
var _ SingleUnarchiver = &ZipArchive{}

// ZipArchive is an extension of an Zip archiver implementing an unarchive method with
// progress feedback
type ZipArchive struct {
	*archiver.Zip
}

// NewZip initializes a new ZipArchive
func NewZip() Unarchiver {
	return Unarchiver{&ZipArchive{archiver.NewZip()}, func(_ string, _ int64, _ bool) {}}
}

func (z *ZipArchive) Ext() string {
	return ".zip"
}

// ExtractNext extracts the next file to destination
func (z *ZipArchive) ExtractNext(destination string) (f archiver.File, err error) {
	f, err = z.Read()
	if err != nil {
		return f, err // don't wrap error; calling loop must break on io.EOF
	}
	defer f.Close()
	header, ok := f.Header.(zip.FileHeader)
	if !ok {
		return f, fmt.Errorf("expected header to be zip.FileHeader but was %T", f.Header)
	}
	return f, z.extractFile(f, filepath.Join(destination, header.Name))
}

func (z *ZipArchive) extractFile(f archiver.File, to string) error {
	// if a directory, no content; simply make the directory and return
	if f.IsDir() {
		return mkdir(to)
	}

	// do not overwrite existing files, if configured
	if !z.OverwriteExisting && fileExists(to) {
		return fmt.Errorf("file already exists: %s", to)
	}

	return writeNewFile(to, f, f.Mode())
}
