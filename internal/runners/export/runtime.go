package export

import (
	"strings"
	"text/template"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/assets"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/runtime"
	"github.com/ActiveState/cli/pkg/runtime_helpers"
)

type Runtime struct {
	prime primeable
	// The remainder is redundant with the above. Refactoring this will follow in a later story so as not to blow
	// up the one that necessitates adding the primer at this level.
	// https://activestatef.atlassian.net/browse/DX-2869
	out       output.Outputer
	analytics analytics.Dispatcher
	svcModel  *model.SvcModel
	auth      *authentication.Auth
	project   *project.Project
	cfg       *config.Instance
}

type RuntimeParams struct {
	Path string
}

func NewRuntime(prime primeable) *Runtime {
	return &Runtime{
		prime,
		prime.Output(),
		prime.Analytics(),
		prime.SvcModel(),
		prime.Auth(),
		prime.Project(),
		prime.Config(),
	}
}

type ErrProjectNotFound struct {
	Path string
}

func (e *ErrProjectNotFound) Error() string {
	return "project not found"
}

func (e *Runtime) Run(params *RuntimeParams) (rerr error) {
	defer rationalizeError(&rerr, e.auth)

	proj := e.project
	if params.Path != "" {
		var err error
		proj, err = project.FromPath(params.Path)
		if err != nil {
			return &ErrProjectNotFound{params.Path}
		}
	}
	if proj == nil {
		return rationalize.ErrNoProject
	}

	e.out.Notice(locale.Tr("export_runtime_statement", proj.NamespaceString(), proj.Dir()))

	rt, err := runtime_helpers.FromProject(proj)
	if err != nil {
		return errs.Wrap(err, "Could not get runtime to export for")
	}

	projectDir := proj.Dir()
	runtimeDir := rt.Path()
	execDir := runtime.ExecutorsPath(runtimeDir)

	contents, err := assets.ReadFileBytes("list_map.tpl")
	if err != nil {
		return errs.Wrap(err, "Could not read asset")
	}
	tmpl, err := template.New("env").Parse(string(contents))
	if err != nil {
		return errs.Wrap(err, "Could not parse env template for output")
	}

	env := rt.Env(false).VariablesWithExecutors

	var envOutput strings.Builder
	err = tmpl.Execute(&envOutput, env)
	if err != nil {
		return errs.Wrap(err, "Could not populate env template for output")
	}

	e.out.Print(output.Prepare(
		locale.Tr("export_runtime_details", projectDir, runtimeDir, execDir, envOutput.String()),
		&struct {
			ProjectDir string            `json:"project"`
			RuntimeDir string            `json:"runtime"`
			ExecDir    string            `json:"executables"`
			Env        map[string]string `json:"environment"`
		}{projectDir, runtimeDir, execDir, env}))

	return nil
}
