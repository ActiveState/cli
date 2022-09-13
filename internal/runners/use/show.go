package use

import (
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
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

type outputFormat struct {
	message   string `locale:message,Message`
	Namespace string `locale:"namespace,Namespace"`
	Path      string `locale:"path,Path"`
}

func (f *outputFormat) MarshalOutput(format output.Format) interface{} {
	if format == output.PlainFormatName {
		return f.message
	}
	return f
}

func (s *Show) Run() error {
	projectDir := s.cfg.GetString(constants.GlobalDefaultPrefname)
	if projectDir == "" {
		return locale.NewInputError("err_use_show_no_default_project", "No default project is set.")
	}

	proj, err := project.FromPath(projectDir)
	if err != nil {
		return locale.WrapError(err, "err_use_show_get_project", "Could not get default project.")
	}

	projectTarget := target.NewProjectTarget(proj, storage.CachePath(), nil, "")

	s.out.Print(&outputFormat{
		locale.Tl("use_show_project_statement", "",
			proj.NamespaceString(),
			projectDir,
			setup.ExecDir(projectTarget.Dir()),
		),
		proj.NamespaceString(),
		projectDir,
	})

	return nil
}
