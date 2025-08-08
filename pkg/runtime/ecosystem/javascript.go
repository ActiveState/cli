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
	packageName := artifact.Name()
	for _, file := range files {
		if file.Name() == "runtime.json" {
			err = injectEnvVar(file.AbsolutePath(), "NPM_CONFIG_PREFIX", "${INSTALLDIR}/usr")
			if err != nil {
				return nil, errs.Wrap(err, "Unable to add NPM_CONFIG_PREFIX to runtime.json")
			}
			continue
		}
		ext := filepath.Ext(file.Name())
		if ext != ".tar.gz" && ext != ".tgz" {
			continue
		}
		if !strings.HasPrefix(file.Name(), artifact.Name()) {
			if i := strings.LastIndex(file.Name(), "-"); i != -1 {
				packageName = file.Name()[:i]
			}
		}
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
	args := []string{"install", "-g", "--offline"} // do not install to current directory
	for _, arg := range e.packages {
		args = append(args, arg)
	}
	env := []string{
		fmt.Sprintf("PATH=%s%s%s", binDir, string(os.PathListSeparator), os.Getenv("PATH")),
		fmt.Sprintf("NPM_CONFIG_PREFIX=%s", filepath.Join(e.runtimePath, "usr")),
	}
	_, stderr, err := osutils.ExecSimple(filepath.Join(binDir, "npm"), args, env)
	if err != nil {
		return errs.Wrap(err, "Error running npm: %s", stderr)
	}
	return nil
}
