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
	Target string // Note: Target paths should always be relative to the archive root, do not use absolute paths
}

func CreateArchive(archive archiver.Writer, archivePath string, workDir string, fileMaps []FileMap) error {
	f, err := os.OpenFile(archivePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return errs.Wrap(err, "Could not create temp file")
	}
	defer f.Close()
	if err := archive.Create(f); err != nil {
		return errs.Wrap(err, "Could not create tar.gz")
	}
	defer archive.Close()

	for _, fileMap := range fileMaps {
		source := fileMap.Source
		if !filepath.IsAbs(source) {
			// Ensure the source path is absolute, because otherwise it will use the global working directory which
			// we're not interested in.
			source = filepath.Join(workDir, source)
		}
		file, err := os.Open(source)
		if err != nil {
			return errs.Wrap(err, "Could not open file")
		}

		fileInfo, err := file.Stat()
		if err != nil {
			return errs.Wrap(err, "Could not stat file")
		}

		// write it to the archive
		err = archive.Write(archiver.File{
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

func CreateTgz(archivePath string, workDir string, fileMaps []FileMap) error {
	return CreateArchive(archiver.NewTarGz(), archivePath, workDir, fileMaps)
}

func CreateZip(archivePath string, workDir string, fileMaps []FileMap) error {
	return CreateArchive(archiver.NewZip(), archivePath, workDir, fileMaps)
}

func FilesWithCommonParent(filepaths ...string) []FileMap {
	var fileMaps []FileMap
	common := fileutils.CommonParentPath(filepaths)
	for _, path := range filepaths {
		path = filepath.ToSlash(path)
		fileMaps = append(fileMaps, FileMap{
			Source: path,
			Target: strings.TrimPrefix(strings.TrimPrefix(path, common), "/"),
		})
	}
	return fileMaps
}
