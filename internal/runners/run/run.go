package run

import (
	"os"
	"strings"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/docker"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/output/txtstyle"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/scriptrun"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/pkg/cmdlets/checker"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/project"
)

// Run contains the run execution context.
type Run struct {
	out      output.Outputer
	proj     *project.Project
	subshell subshell.SubShell
	cfg      *config.Instance
}

type primeable interface {
	primer.Outputer
	primer.Projecter
	primer.Subsheller
	primer.Configurer
}

// New constructs a new instance of Run.
func New(prime primeable) *Run {
	return &Run{
		prime.Output(),
		prime.Project(),
		prime.Subshell(),
		prime.Config(),
	}
}

// Run runs the Run run runner.
func (r *Run) Run(name string, args []string) error {
	logging.Debug("Execute")

	if r.proj == nil {
		return locale.NewInputError("err_no_project")
	}

	if name == "" {
		return locale.NewError("error_state_run_undefined_name")
	}

	r.out.Notice(txtstyle.NewTitle(locale.Tl("run_script_title", "Running Script: [ACTIONABLE]{{.V0}}[/RESET]", name)))

	if authentication.Get().Authenticated() {
		checker.RunCommitsBehindNotifier(r.proj, r.out)
	}

	script := r.proj.ScriptByName(name)
	if script == nil {
		return locale.NewInputError("error_state_run_unknown_name", "Script does not exist: {{.V0}}", name)
	}

	img := script.Image()
	if img == nil && r.proj.Config().Image() != nil {
		img = r.proj.Config().Image()
	}
	if img != nil && r.proj.CommitID() != os.Getenv(constants.DockerLabelCommitEnvVarName) {
		return r.runViaDocker(script)
	}

	scriptrunner := scriptrun.New(r.out, r.subshell, r.proj, r.cfg)
	if !script.Standalone() && scriptrunner.NeedsActivation() {
		if err := scriptrunner.PrepareVirtualEnv(); err != nil {
			return locale.WrapError(err, "err_script_run_preparevenv", "Could not prepare virtual environment.")
		}
	}

	if len(script.Languages()) == 0 {
		r.out.Notice(output.Heading(locale.Tl("deprecation_warning", "Deprecation Warning!")))
		r.out.Notice(locale.Tl(
			"run_warn_deprecated_script_without_language",
			"Scripts without a defined language currently fall back to using the default shell for your platform. This fallback mechanic will soon stop working and a language will need to be explicitly defined for each script. Please configure the '[ACTIONABLE]language[/RESET]' field with a valid option (one of [ACTIONABLE]{{.V0}}[/RESET])",
			strings.Join(language.RecognizedNames(), ", ")))
	}

	r.out.Notice(output.Heading(locale.Tl("script_output", "Script Output")))
	return scriptrunner.Run(script, args)
}

type writer struct {
	out output.Outputer
}

func (w *writer) Write(p []byte) (n int, err error) {
	w.out.Print(p)
	return len(p), nil
}

func (r *Run) runViaDocker(script *project.Script) error {
	if script.Image() == nil {
		return errs.New("Script image is nil")
	}

	r.out.Notice(output.Heading(locale.Tl("run_script_docker", "Launching Docker Container: [ACTIONABLE]{{.V0}}[/RESET]", script.Image().Name())))

	// Run docker container
	opts := docker.TargetOptions{
		Image:    script.Image().Name(),
		Commit:   r.proj.CommitID(),
		YamlPath: r.proj.Source().Path(),
	}
	builder, err := docker.NewBuilder(opts)
	if err != nil {
		return errs.Wrap(err, "Could not create builder")
	}
	if !builder.TargetImageExists() {
		if err := builder.Build(&writer{r.out}); err != nil {
			return locale.WrapError(err, "err_run_build_docker", "Could not build Docker image.")
		}
	}
	runner, err := docker.NewRunner(opts)
	if err != nil {
		return errs.Wrap(err, "Could not create runner")
	}
	cmd := os.Args
	cmd[0] = constants.CommandName
	cmd = append(cmd, "--output=simple")
	if err := runner.Run(cmd, &writer{r.out}); err != nil {
		return locale.WrapError(err, "err_run_build_docker", "Could not run Docker container.")
	}

	return nil
}