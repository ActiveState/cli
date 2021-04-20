// +build windows

package legacyupd

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"

	"github.com/ActiveState/cli/internal/logging"
)

func (u *Updater) fetchBin(file []byte) ([]byte, error) {
	buf := new(bytes.Buffer)

	files, err := zip.NewReader(bytes.NewReader(file), int64(len(file)))
	if len(files.File) == 0 {
		return nil, errors.New("Update zip contains no files")
	}
	filer, err := files.File[0].Open()
	if err != nil {
		logging.Error(err.Error())
		return nil, err
	}
	if _, err = io.Copy(buf, filer); err != nil {
		logging.Error(err.Error())
		return nil, err
	}

	return buf.Bytes(), nil
}
