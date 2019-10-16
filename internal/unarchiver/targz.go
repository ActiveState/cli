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
)

/*
  This file implements an extension for the `archiver.TarGz` type that unarchives a tar.gz archive
  reporting its progress.
*/

// ensure that it implements the SingleUnarchiver interface
var _ SingleUnarchiver = &TarGzArchive{}

// TarGzArchive is an extension of an TarGz archiver implementing an unarchive method with
// progress feedback
type TarGzArchive struct {
	archiver.TarGz
}

// NewTarGz initializes a new TarGzArchiver
func NewTarGz() Unarchiver {
	return Unarchiver{&TarGzArchive{*archiver.DefaultTarGz}}
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

// ExtractNext extracts the next file to destination
func (ar *TarGzArchive) ExtractNext(destination string) (f archiver.File, err error) {
	f, err = ar.Read()
	if err != nil {
		return f, err // don't wrap error; calling loop must break on io.EOF
	}
	header, ok := f.Header.(*tar.Header)
	if !ok {
		return f, fmt.Errorf("expected header to be *tar.Header but was %T", f.Header)
	}
	return f, ar.untarFile(f, filepath.Join(destination, header.Name))
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
