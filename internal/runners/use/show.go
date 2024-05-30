package use

import (
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runbits/runtime/target"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type Show struct {
	out output.Outputer
	cfg *config.Instance
}

func NewShow(prime primeable) *Show {
	return &Show{
		prime.Output(),
		prime.Config(),
	}
}

func (s *Show) Run() error {
	projectDir := s.cfg.GetString(constants.GlobalDefaultPrefname)
	if projectDir == "" {
		return locale.NewInputError("err_use_show_no_default_project", "No project is being used.")
	}

	proj, err := project.FromPath(projectDir)
	if err != nil {
		if errs.Matches(err, &projectfile.ErrorNoProject{}) {
			return locale.WrapError(err, "err_use_default_project_does_not_exist")
		}
		return locale.WrapError(err, "err_use_show_get_project", "Could not get your project.")
	}

	projectTarget := target.NewProjectTarget(proj, nil, "")
	executables := setup.ExecDir(projectTarget.Dir())

	s.out.Print(output.Prepare(
		locale.Tr("use_show_project_statement",
			proj.NamespaceString(),
			projectDir,
			executables,
		),
		&struct {
			Namespace   string `json:"namespace"`
			Path        string `json:"path"`
			Executables string `json:"executables"`
		}{
			proj.NamespaceString(),
			projectDir,
			executables,
		},
	))

	return nil
}
