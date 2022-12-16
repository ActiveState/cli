package unarchiver

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"

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
