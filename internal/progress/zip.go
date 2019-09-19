package progress

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ActiveState/archiver"
)

/*
  This file implements an extension of the `archiver.Zip` type that unarchives a zip archive
  reporting its progress.
*/

// ensure that it implements the ProgressUnarchiver interface
var _ Unarchiver = &TarGzArchiveReader{}

// ZipArchiveReader is an extension of an Zip archiver implementing an unarchive method with
// progress feedback
type ZipArchiveReader struct {
	archiver.Zip
}

// UnarchiveWithProgress unpacks the .zip file at source to destination.
// Destination will be treated as a folder name.
// progressIncrement will be called after each unpacked file with the size of that file in bytes
func (z *ZipArchiveReader) UnarchiveWithProgress(source, destination string, progressIncrement func(int64)) error {
	if !fileExists(destination) && z.MkdirAll {
		err := mkdir(destination)
		if err != nil {
			return fmt.Errorf("preparing destination: %v", err)
		}
	}

	file, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("opening source file: %v", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("statting source file: %v", err)
	}

	err = z.Open(file, fileInfo.Size())
	if err != nil {
		return fmt.Errorf("opening zip archive for reading: %v", err)
	}
	defer z.Close()

	for {
		f, err := z.extractNext(destination)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading file in zip archive: %v", err)
		}
		progressIncrement(f.Size())
	}

	return nil
}

func (z *ZipArchiveReader) extractNext(to string) (archiver.File, error) {
	f, err := z.Read()
	if err != nil {
		return f, err // don't wrap error; calling loop must break on io.EOF
	}
	defer f.Close()
	header, ok := f.Header.(zip.FileHeader)
	if !ok {
		return f, fmt.Errorf("expected header to be zip.FileHeader but was %T", f.Header)
	}
	return f, z.extractFile(f, filepath.Join(to, header.Name))
}

func (z *ZipArchiveReader) extractFile(f archiver.File, to string) error {
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
