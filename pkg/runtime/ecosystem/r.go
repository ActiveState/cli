package ecosystem

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/osutils"

	"github.com/ActiveState/cli/pkg/buildplan"
)

const rLibraryDir = "usr/lib/R/library"

type R struct {
	runtimePath     string
	installPackages []string
}

func (e *R) Init(runtimePath string, buildplan *buildplan.BuildPlan) error {
	e.runtimePath = runtimePath
	e.installPackages = []string{}
	return nil
}

func (e *R) Namespaces() []string {
	return []string{"language/R"}
}

func (e *R) Add(artifact *buildplan.Artifact, artifactSrcPath string) ([]string, error) {
	files, err := fileutils.ListDir(artifactSrcPath, false)
	if err != nil {
		return nil, errs.Wrap(err, "Unable to read artifact source directory")
	}
	packageName := artifact.Name()
	for _, file := range files {
		ext := filepath.Ext(file.Name())
		if ext != ".gz" && ext != ".tgz" && ext != ".zip" {
			continue
		}
		e.installPackages = append(e.installPackages, file.AbsolutePath())
	}
	installedDir := filepath.Join(rLibraryDir, packageName) // Apply() will install here
	return []string{installedDir}, nil
}

func (e *R) Remove(name, version string, installedFiles []string) (rerr error) {
	for _, dir := range installedFiles {
		if !fileutils.DirExists(dir) {
			continue
		}
		err := os.RemoveAll(dir)
		if err != nil {
			rerr = errs.Pack(rerr, errs.Wrap(err, "Unable to remove directory for '%s': %s", name, dir))
		}
	}
	return rerr
}

func (e *R) Apply() error {
	if len(e.installPackages) == 0 {
		return nil // nothing to do
	}

	binDir := filepath.Join(e.runtimePath, "usr", "bin")
	tgzs := []string{}
	for _, tgz := range e.installPackages {
		tgzs = append(tgzs, fmt.Sprintf(`"%s"`, tgz))
	}
	args := []string{
		"-e",
		fmt.Sprintf("install.packages(c(%s), lib='%s', repos=NULL)",
			strings.Join(tgzs, ","), filepath.Join(e.runtimePath, rLibraryDir)),
	}
	env := []string{}

	_, stderr, err := osutils.ExecSimple(filepath.Join(binDir, "R"), args, env)
	if err != nil {
		return errs.Wrap(err, "Error running R: %s", stderr)
	}

	return nil
}
