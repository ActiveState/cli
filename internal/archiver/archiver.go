package archiver

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/mholt/archives"
)

type FileMap struct {
	Source string
	Target string // Note: Target paths should always be relative to the archive root, do not use absolute paths
}

func CreateArchive(format archives.CompressedArchive, archivePath string, workDir string, fileMaps []FileMap) error {
	f, err := os.OpenFile(archivePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return errs.Wrap(err, "Could not create temp file")
	}
	defer f.Close()

	filesMap := make(map[string]string)
	for _, fileMap := range fileMaps {
		source := fileMap.Source
		if !filepath.IsAbs(source) {
			// Ensure the source path is absolute, because otherwise it will use the global working directory which
			// we're not interested in.
			source = filepath.Join(workDir, source)
		}
		filesMap[source] = fileMap.Target
	}

	ctx := context.Background()
	files, err := archives.FilesFromDisk(ctx, nil, filesMap)
	if err != nil {
		return errs.Wrap(err, "Could not create file info structs")
	}

	err = format.Archive(ctx, f, files)
	if err != nil {
		return errs.Wrap(err, "Could not create archive")
	}

	return nil
}

func CreateTgz(archivePath string, workDir string, fileMaps []FileMap) error {
	return CreateArchive(archives.CompressedArchive{
		Compression: archives.Gz{},
		Archival:    archives.Tar{},
	}, archivePath, workDir, fileMaps)
}

func CreateZip(archivePath string, workDir string, fileMaps []FileMap) error {
	return CreateArchive(archives.CompressedArchive{
		Archival: archives.Zip{},
	}, archivePath, workDir, fileMaps)
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
