package deploy

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/virtualenvironment"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

var (
	FailNoCommitForProject = failures.Type("deploy.fail.nocommit")
)

type Params struct {
	Namespace project.Namespaced
	Path      string
	Step      Step
}

type Deploy struct {
	output output.Outputer

	DefaultBranchForProjectName DefaultBranchForProjectNameFunc
	NewRuntimeInstaller         NewInstallerFunc
}

func NewDeploy(out output.Outputer) *Deploy {
	return &Deploy{
		out,
		model.DefaultBranchForProjectName,
		NewInstaller,
	}
}

func (d *Deploy) Run(params *Params) error {
	installer, err := d.createInstaller(params.Namespace, params.Path)
	if err != nil {
		return err
	}

	return runSteps(installer, params.Step)
}

func (d *Deploy) createInstaller(namespace project.Namespaced, path string) (Installable, *failures.Failure) {
	branch, fail := d.DefaultBranchForProjectName(namespace.Owner, namespace.Project)
	if fail != nil {
		return nil, fail
	}

	if branch.CommitID == nil {
		return nil, FailNoCommitForProject.New(locale.Tr("err_deploy_no_commits", namespace.String()))
	}

	return d.NewRuntimeInstaller(*branch.CommitID, namespace.Owner, namespace.Project, path)
}

func runSteps(installer Installable, step Step) error {
	logging.Debug("runSteps: %s", step.String())

	if step == UnsetStep || step == InstallStep {
		logging.Debug("Running install step")
		_, fail := installer.Install()
		if fail != nil {
			return fail
		}
	}
	if step == UnsetStep || step == ConfigureStep {
		logging.Debug("Running configure step")
		if err := configure(installer); err != nil {
			return err
		}
	}

	return nil
}

func configure(installer Installable) error {
	installDirs, fail := installer.InstallDirs()
	if fail != nil {
		return fail
	}

	sshell, fail := subshell.Get()
	if fail != nil {
		return fail
	}

	venv := virtualenvironment.NewWithArtifacts(installDirs)
	env := venv.GetEnv(false, "")

	return sshell.WriteUserEnv(env)
}

func (d *Deploy) report() error {
	return nil
}
