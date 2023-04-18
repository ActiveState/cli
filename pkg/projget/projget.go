package projget

import (
	"github.com/ActiveState/cli/internal/constraints"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/ActiveState/cli/pkg/projectfile/vars"
)

func NewProject(out output.Outputer, auth *authentication.Auth, shell string) (*project.Project, error) {
	// Retrieve project file
	pjPath, err := projectfile.GetProjectFilePath()
	if err != nil && errs.Matches(err, &projectfile.ErrorNoProjectFromEnv{}) {
		// Fail if we are meant to inherit the projectfile from the environment, but the file doesn't exist
		return nil, err
	}

	// Set up project (if we have a valid path)
	var pj *project.Project
	if pjPath != "" {
		pjf, err := projectfile.FromPath(pjPath)
		if err != nil {
			return nil, err
		}
		pj, err = project.New(pjf, out)
		if err != nil {
			return nil, err
		}
	}

	if pj != nil {
		registerProjectVars := func() {
			projVars := vars.New(auth, vars.NewProject(pj), shell)
			conditional := constraints.NewPrimeConditional(projVars)
			project.RegisterConditional(conditional)
			_ = project.RegisterStruct(projVars)
		}

		pj.SetUpdateCallback(registerProjectVars)
		registerProjectVars()
	}

	return pj, nil
}

func NewProjectForTest() (*project.Project, error) {
	return NewProject(output.Get(), nil, "noshell")
}
