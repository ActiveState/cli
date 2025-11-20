package unarchiver

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"path/filepath"

	"github.com/ActiveState/cli/internal/errs"
)

type TarGzBlob struct {
	blob []byte
}

func NewTarGzBlob(blob []byte) *TarGzBlob {
	return &TarGzBlob{blob}
}

func (t *TarGzBlob) Unarchive(dest string) error {
	unzipped, err := gzip.NewReader(bytes.NewReader(t.blob))
	if err != nil {
		return errs.Wrap(err, "Could not read tar.gz archive")
	}

	reader := tar.NewReader(unzipped)

	for {
		hdr, err := reader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return errs.Wrap(err, "Failed to read file from archive.")
		}
		data := io.LimitReader(reader, hdr.Size)

		err = untarSingleFile(hdr, data, dest, hdr.Name, false)
		if err != nil {
			return errs.Wrap(err, "Failed to unpack file %s from tar archive", hdr.Name)
		}
	}

	return nil
}

func untarSingleFile(hdr *tar.Header, data io.Reader, destination, relTo string, overwriteExisting bool) error {
	to := filepath.Join(destination, relTo)
	// do not overwrite existing files, if configured
	if !hdr.FileInfo().IsDir() && !overwriteExisting && fileExists(to) {
		return fmt.Errorf("file already exists: %s", to)
	}

	switch hdr.Typeflag {
	case tar.TypeDir:
		return mkdir(to)
	case tar.TypeReg, tar.TypeChar, tar.TypeBlock, tar.TypeFifo:
		return writeNewFile(to, data, hdr.FileInfo().Mode())
	case tar.TypeSymlink:
		return writeNewSymbolicLink(to, hdr.Linkname)
	case tar.TypeLink:
		// hard links are always relative to the destination directory
		link := filepath.Join(destination, hdr.Linkname)
		return writeNewHardLink(to, link)
	case tar.TypeXGlobalHeader:
		return nil // ignore the pax global header from git-generated tarballs
	default:
		return fmt.Errorf("%s: unknown type flag: %c", hdr.Name, hdr.Typeflag)
	}
}
