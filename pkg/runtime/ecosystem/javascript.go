package ecosystem

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/osutils"

	"github.com/ActiveState/cli/pkg/buildplan"
)

const nodeModulesDir = "usr/lib/node_modules"

type JavaScript struct {
	runtimePath string
	packages    []string
}

func (e *JavaScript) Init(runtimePath string, buildplan *buildplan.BuildPlan) error {
	e.runtimePath = runtimePath
	e.packages = []string{}
	return nil
}

func (e *JavaScript) Namespaces() []string {
	return []string{"language/javascript"}
}

func (e *JavaScript) Add(artifact *buildplan.Artifact, artifactSrcPath string) ([]string, error) {
	files, err := fileutils.ListDir(artifactSrcPath, false)
	if err != nil {
		return nil, errs.Wrap(err, "Unable to read artifact source directory")
	}
	var packageName string
	for _, file := range files {
		ext := filepath.Ext(file.Name())
		if ext != ".tar.gz" && ext != ".tgz" {
			continue
		}
		packageName = file.Name()[:len(file.Name())-len(ext)]
		e.packages = append(e.packages, file.AbsolutePath())
	}
	installedDir := filepath.Join(nodeModulesDir, packageName) // Apply() will install here
	return []string{installedDir}, nil
}

func (e *JavaScript) Remove(artifact *buildplan.Artifact) error {
	return nil // TODO: CP-956
}

func (e *JavaScript) Apply() error {
	if len(e.packages) == 0 {
		return nil // nothing to do
	}

	binDir := filepath.Join(e.runtimePath, "usr", "bin")
	args := []string{"install", "-g", "--offline"}
	for _, arg := range e.packages {
		args = append(args, arg)
	}
	env := []string{
		fmt.Sprintf("PATH=%s%s%s", binDir, string(os.PathListSeparator), os.Getenv("PATH")),
	}
	_, stderr, err := osutils.ExecSimple(filepath.Join(binDir, "npm"), args, env)
	if err != nil {
		return errs.Wrap(err, "Error running npm: %s", stderr)
	}
	return nil
}
