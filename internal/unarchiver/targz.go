package unarchiver

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/ActiveState/archiver"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/progress"
)

/*
  This file implements an extension for the `archiver.TarGz` type that unarchives a tar.gz archive
  reporting its progress.
*/

// ensure that it implements the ProgressUnarchiver interface
var _ Unarchiver = &TarGzArchive{}

// TarGzArchive is an extension of an TarGz archiver implementing an unarchive method with
// progress feedback
type TarGzArchive struct {
	archiver.TarGz
	inputStreamWrapper func(io.Reader) *io.Reader
}

// NewTarGz initializes a new TarGzArchiver
func NewTarGz() *TarGzArchive {
	return &TarGzArchive{*archiver.DefaultTarGz, func(r io.Reader) *io.Reader { return &r }}
}

// SetInputStreamWrapper sets a new wrapper function for the io Reader used during unpacking
func (ar *TarGzArchive) SetInputStreamWrapper(f func(io.Reader) *io.Reader) {
	ar.inputStreamWrapper = f
}

// GetExtractedSize returns the size of the extracted summed up files in the archive
func (ar *TarGzArchive) GetExtractedSize(source string) (int, error) {
	archiveStream, err := os.Open(source)
	if err != nil {
		return 0, err
	}
	defer archiveStream.Close()
	var size int
	buf := make([]byte, 10*1024)

	gzr, err := gzip.NewReader(archiveStream)
	if err != nil {
		return 0, err
	}
	defer gzr.Close()

	for {
		nread, err := gzr.Read(buf)
		if err == io.EOF {
			return size, nil
		}
		if err != nil {
			return 0, err
		}
		size += nread
	}

}

// UnarchiveWithProgress unpacks the files from the source directory into the destination directory
// After a file is unpacked, the callback is called
func (ar *TarGzArchive) UnarchiveWithProgress(source, destination string, fn progress.FileSizeCallback) error {
	if !fileExists(destination) && ar.MkdirAll {
		err := mkdir(destination)
		if err != nil {
			return fmt.Errorf("preparing destination: %v", err)
		}
	}

	archiveFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer archiveFile.Close()

	wrappedStream := ar.inputStreamWrapper(archiveFile)

	// read one file at a time from the archive
	err = ar.Open(*wrappedStream, 0)
	if err != nil {
		return err
	}
	// note: that this is obviously not thread-safe
	defer ar.Close()

	for {
		f, err := ar.untarNext(destination)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// calling the increment callback
		logging.Debug("Extracted %s File size: %d", f.Name(), f.Size())
		fn(int(f.Size()))
	}
	return nil
}

func (ar *TarGzArchive) untarNext(to string) (archiver.File, error) {
	f, err := ar.Read()
	if err != nil {
		return f, err // don't wrap error; calling loop must break on io.EOF
	}
	header, ok := f.Header.(*tar.Header)
	if !ok {
		return f, fmt.Errorf("expected header to be *tar.Header but was %T", f.Header)
	}
	return f, ar.untarFile(f, filepath.Join(to, header.Name))
}

func (ar *TarGzArchive) untarFile(f archiver.File, to string) error {
	// do not overwrite existing files, if configured
	if !f.IsDir() && !ar.OverwriteExisting && fileExists(to) {
		return fmt.Errorf("file already exists: %s", to)
	}

	hdr, ok := f.Header.(*tar.Header)
	if !ok {
		return fmt.Errorf("expected header to be *tar.Header but was %T", f.Header)
	}

	switch hdr.Typeflag {
	case tar.TypeDir:
		return mkdir(to)
	case tar.TypeReg, tar.TypeRegA, tar.TypeChar, tar.TypeBlock, tar.TypeFifo:
		return writeNewFile(to, f, f.Mode())
	case tar.TypeSymlink:
		return writeNewSymbolicLink(to, hdr.Linkname)
	case tar.TypeLink:
		// NOTE: this is a hack that fixes an issue for choosing the correct path to the old file
		// that is being linked to. This fix will only address calls to Unarchive, not Extract and
		// is generally only known to be useful for ActiveState, at the moment.
		return writeNewHardLink(to, path.Join(path.Dir(to), path.Base(hdr.Linkname)))
	case tar.TypeXGlobalHeader:
		return nil // ignore the pax global header from git-generated tarballs
	default:
		return fmt.Errorf("%s: unknown type flag: %c", hdr.Name, hdr.Typeflag)
	}
}
