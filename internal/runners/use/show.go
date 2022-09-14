package use

import (
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runbits/findproject"
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

	proj, err := findproject.FromPath(projectDir, nil)
	if err != nil {
		if findproject.IsLocalProjectDoesNotExistError(err) {
			return locale.WrapError(err,
				"err_use_show_default_project_does_not_exist",
				"The default project no longer exists. Please either check it out again or run [ACTIONABLE]state use reset[/RESET].")
		}
		return locale.WrapError(err, "err_use_show_get_project", "Could not get default project.")
	}

	s.out.Print(&outputFormat{
		locale.Tl("use_show", "The default project to use is {{.V0}}, located at {{.V1}}",
			proj.NamespaceString(),
			projectDir,
		),
		proj.NamespaceString(),
		projectDir,
	})

	return nil
}
