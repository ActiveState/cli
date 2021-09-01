package virtualenv

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/virtualenvironment"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/project"
)

type VirtualEnv struct {
	out  output.Outputer
	proj *project.Project
	*virtualenvironment.VirtualEnvironment
}

func New(out output.Outputer, proj *project.Project, runtime *runtime.Runtime) *VirtualEnv {
	return &VirtualEnv{
		out:                out,
		proj:               proj,
		VirtualEnvironment: virtualenvironment.New(runtime),
	}
}

func (v *VirtualEnv) GetEnv(inherit bool, useExecutors bool, projectDir string) (map[string]string, error) {
	v.out.Notice(locale.Tl(
		"virtualenv_creation",
		"Creating a virtual environment for {{.V0}}.",
		v.proj.Name(),
	))

	return v.VirtualEnvironment.GetEnv(inherit, useExecutors, projectDir)
}
