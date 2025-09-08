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
	runtimePath       string
	installPackages   []string
	uninstallPackages []string
}

func (e *JavaScript) Init(runtimePath string, buildplan *buildplan.BuildPlan) error {
	e.runtimePath = runtimePath
	e.installPackages = []string{}
	e.uninstallPackages = []string{}
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
		if ext != ".gz" && ext != ".tgz" {
			continue
		}
		if !strings.HasPrefix(file.Name(), packageName) {
			// A vast majority of the time, the installed package name is the artifact name.
			// When this is not the case, extract the package name from the file name. The file name is
			// of the form <name>-<version>.tar.gz, so extract the <name> part.
			if i := strings.LastIndex(file.Name(), "-"); i != -1 {
				packageName = file.Name()[:i]
			}
		}
		e.installPackages = append(e.installPackages, file.AbsolutePath())
	}
	installedDir := filepath.Join(nodeModulesDir, packageName) // Apply() will install here
	return []string{installedDir}, nil
}

func (e *JavaScript) Remove(name, version string, installedFiles []string) error {
	e.uninstallPackages = append(e.uninstallPackages, name)
	return nil
}

func (e *JavaScript) Apply() error {
	if len(e.installPackages) == 0 && len(e.uninstallPackages) == 0 {
		return nil // nothing to do
	}

	binDir := filepath.Join(e.runtimePath, "usr", "bin")
	installArgs := []string{"install", "-g", "--offline"} // do not install to current directory
	for _, arg := range e.installPackages {
		installArgs = append(installArgs, arg)
	}
	uninstallArgs := []string{"uninstall", "-g", "--no-save"} // do not remove from current directory
	for _, arg := range e.uninstallPackages {
		uninstallArgs = append(uninstallArgs, arg)
	}
	env := []string{
		fmt.Sprintf("PATH=%s%s%s", binDir, string(os.PathListSeparator), os.Getenv("PATH")),
		fmt.Sprintf("NPM_CONFIG_PREFIX=%s", filepath.Join(e.runtimePath, "usr")),
	}

	if len(e.installPackages) > 0 {
		_, stderr, err := osutils.ExecSimple(filepath.Join(binDir, "npm"), installArgs, env)
		if err != nil {
			return errs.Wrap(err, "Error running npm install: %s", stderr)
		}
	}

	if len(e.uninstallPackages) > 0 {
		_, stderr, err := osutils.ExecSimple(filepath.Join(binDir, "npm"), uninstallArgs, env)
		if err != nil {
			return errs.Wrap(err, "Error running npm uninstall: %s", stderr)
		}
	}
	return nil
}
