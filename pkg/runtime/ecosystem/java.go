package ecosystem

import (
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"

	"github.com/ActiveState/cli/pkg/buildplan"
)

type Java struct {
	libDir string
}

func (e *Java) Init(runtimePath string, buildplan *buildplan.BuildPlan) error {
	e.libDir = filepath.Join(runtimePath, "lib")
	err := fileutils.MkdirUnlessExists(e.libDir)
	if err != nil {
		return errs.Wrap(err, "Unable to create runtime lib directory")
	}
	return nil
}

func (e *Java) Namespaces() []string {
	return []string{"language/java"}
}

func (e *Java) Add(artifact *buildplan.Artifact, artifactSrcPath string) ([]string, error) {
	installedFiles := []string{}
	files, err := fileutils.ListDir(artifactSrcPath, false)
	if err != nil {
		return nil, errs.Wrap(err, "Unable to read artifact source directory")
	}
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".jar") {
			continue
		}
		installedFile := filepath.Join(e.libDir, file.Name())
		err = fileutils.CopyFile(file.AbsolutePath(), installedFile)
		if err != nil {
			return nil, errs.Wrap(err, "Unable to copy artifact jar into runtime lib directory")
		}
		installedFiles = append(installedFiles, installedFile)
	}
	return installedFiles, nil
}

func (e *Java) Remove(artifact *buildplan.Artifact) error {
	return nil // TODO: CP-956
}

func (e *Java) Apply() error {
	return nil
}
