package unarchiver

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/errs"
)

type ZipBlob struct {
	blob []byte
}

func NewZipBlob(blob []byte) *ZipBlob {
	return &ZipBlob{blob}
}

func (z *ZipBlob) Unarchive(dest string) error {
	reader, err := zip.NewReader(bytes.NewReader(z.blob), int64(len(z.blob)))
	if err != nil {
		return errs.Wrap(err, "Could not read zip archive")
	}

	for _, f := range reader.File {
		if err := z.unzipFile(f, dest); err != nil {
			return errs.Wrap(err, "Failed to unzip file: %s", f.Name)
		}
	}
	return nil
}

func (z *ZipBlob) unzipFile(file *zip.File, dest string) error {
	rc, err := file.Open()
	if err != nil {
		return errs.Wrap(err, "Could not read file in archive: %s", file.Name)
	}
	defer rc.Close()

	fpath := filepath.Join(dest, file.Name)
	if file.FileInfo().IsDir() {
		if err := os.MkdirAll(fpath, file.Mode()); err != nil {
			return errs.Wrap(err, "Could not create dir: %s", fpath)
		}
		return nil
	}

	if err := writeNewFile(fpath, rc, file.Mode()); err != nil {
		return errs.Wrap(err, "Could write file %s", fpath)
	}

	return nil
}
