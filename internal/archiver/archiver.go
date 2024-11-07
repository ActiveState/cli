package archiver

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/mholt/archiver/v3"
)

type FileMap struct {
	Source string
	Target string
}

func CreateTgz(filepath string, fileMaps []FileMap) error {
	f, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return errs.Wrap(err, "Could not create temp file")
	}
	defer f.Close()
	tgz := archiver.NewTarGz()
	if err := tgz.Create(f); err != nil {
		return errs.Wrap(err, "Could not create tar.gz")
	}
	defer tgz.Close()

	for _, fileMap := range fileMaps {
		file, err := os.Open(fileMap.Source)
		if err != nil {
			return errs.Wrap(err, "Could not open file")
		}

		fileInfo, err := file.Stat()
		if err != nil {
			return errs.Wrap(err, "Could not stat file")
		}

		// write it to the archive
		err = tgz.Write(archiver.File{
			FileInfo: archiver.FileInfo{
				FileInfo:   fileInfo,
				CustomName: fileMap.Target,
			},
			ReadCloser: file,
		})
		file.Close()
		if err != nil {
			return errs.Wrap(err, "Could not write file to tar.gz")
		}
	}

	return nil
}

func FilesWithCommonParent(filepaths ...string) []FileMap {
	var fileMaps []FileMap
	common := fileutils.CommonParentPath(filepaths)
	for _, path := range filepaths {
		path = filepath.ToSlash(path)
		fileMaps = append(fileMaps, FileMap{
			Source: filepath.ToSlash(path),
			Target: strings.TrimPrefix(filepath.ToSlash(strings.TrimPrefix(path, common)), "/"),
		})
	}
	return fileMaps
}
