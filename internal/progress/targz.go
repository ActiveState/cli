package progress

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/ActiveState/archiver"
	"github.com/ActiveState/cli/internal/logging"
)

/*
  This file implements an extension for the `archiver.TarGz` type that unarchives a tar.gz archive
  reporting its progress.
*/

// ensure that it implements the ProgressUnarchiver interface
var _ Unarchiver = &TarGzArchiveReader{}

// TarGzArchiveReader is an extension of an TarGz archiver implementing an unarchive method with
// progress feedback
type TarGzArchiveReader struct {
	archiver.TarGz
}

// UnarchiveWithProgress unpacks the files from the source directory into the destination directory
// After a file is unpacked, the progressIncrement callback is called
func (ar *TarGzArchiveReader) UnarchiveWithProgress(source, destination string, progressIncrement func(int64)) error {
	if !fileExists(destination) && ar.MkdirAll {
		err := mkdir(destination)
		if err != nil {
			return fmt.Errorf("preparing destination: %v", err)
		}
	}

	archiveStream, err := os.Open(source)
	if err != nil {
		return err
	}
	defer archiveStream.Close()

	// read one file at a time from the archive
	err = ar.Open(archiveStream, 0)
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
		progressIncrement(f.Size())
	}
	return nil
}

func (ar *TarGzArchiveReader) untarNext(to string) (archiver.File, error) {
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

func (ar *TarGzArchiveReader) untarFile(f archiver.File, to string) error {
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
