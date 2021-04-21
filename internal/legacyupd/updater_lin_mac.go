// +build !windows

package legacyupd

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"

	"github.com/ActiveState/cli/internal/logging"
)

func (u *Updater) fetchBin(file []byte) ([]byte, error) {
	buf := new(bytes.Buffer)

	filer, err := gzip.NewReader(bytes.NewReader(file))
	if err != nil {
		logging.Error(err.Error())
		return nil, err
	}
	if _, err = io.Copy(buf, filer); err != nil {
		logging.Error(err.Error())
		return nil, err
	}

	return u.fetchFileFromTar(buf.Bytes())
}

func (u *Updater) fetchFileFromTar(file []byte) ([]byte, error) {
	buf := new(bytes.Buffer)

	filer := tar.NewReader(bytes.NewReader(file))
	filer.Next()

	if _, err := io.Copy(buf, filer); err != nil {
		logging.Error(err.Error())
		return nil, err
	}

	return buf.Bytes(), nil
}
