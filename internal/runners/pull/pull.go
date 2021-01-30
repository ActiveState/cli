package pull

import (
	"os"
	"path"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/hail"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
)

type Pull struct {
	prompt  prompt.Prompter
	project *project.Project
	out     output.Outputer
	cfg     *config.Instance
}

type PullParams struct {
	Force      bool
	SetProject string
}

type primeable interface {
	primer.Prompter
	primer.Projecter
	primer.Outputer
	primer.Configurer
}

func New(prime primeable) *Pull {
	return &Pull{
		prime.Prompt(),
		prime.Project(),
		prime.Output(),
		prime.Config(),
	}
}

type outputFormat struct {
	Message string `locale:"message,Message"`
	Success bool   `locale:"success,Success"`
}

func (f *outputFormat) MarshalOutput(format output.Format) interface{} {
	switch format {
	case output.EditorV0FormatName:
		return f.editorV0Format()
	case output.PlainFormatName:
		return f.Message
	}

	return f
}

func (p *Pull) Run(params *PullParams) error {
	if p.project == nil {
		return locale.NewInputError("err_no_project")
	}

	if p.project.IsHeadless() && params.SetProject == "" {
		return locale.NewInputError("err_pull_headless", "You must first create a project. Please visit {{.V0}} to create your project.", p.project.URL())
	}

	// Determine the project to pull from
	target, err := targetProject(p.project, params.SetProject)
	if err != nil {
		return errs.Wrap(err, "Unable to determine target project")
	}

	if params.SetProject != "" {
		related, err := areCommitsRelated(*target.CommitID, p.project.CommitUUID())
		if !related && !params.Force {
			confirmed, err := p.prompt.Confirm(locale.T("confirm"), locale.Tl("confirm_unrelated_pull_set_project", "If you switch to {{.V0}}, you may lose changes to your project. Are you sure you want to do this?", target.String()), new(bool))
			if err != nil {
				return locale.WrapError(err, "err_pull_confirm", "Failed to get user confirmation to update project")
			}
			if !confirmed {
				return locale.NewInputError("err_pull_aborted", "Pull aborted by user")
			}
		}

		err = p.project.Source().SetNamespace(target.Owner, target.Project)
		if err != nil {
			return locale.WrapError(err, "err_pull_update_namespace", "Cannot update the namespace in your project file.")
		}
	}

	// Update the commit ID in the activestate.yaml
	if p.project.CommitID() != target.CommitID.String() {
		err := p.project.Source().SetCommit(target.CommitID.String(), false)
		if err != nil {
			return locale.WrapError(err, "err_pull_update", "Cannot update the commit in your project file.")
		}

		p.out.Print(&outputFormat{
			locale.Tr("pull_updated", target.String(), target.CommitID.String()),
			true,
		})
	} else {
		p.out.Print(&outputFormat{
			locale.Tl("pull_not_updated", "Your activestate.yaml is already up to date."),
			false,
		})
	}

	actID := os.Getenv(constants.ActivatedStateIDEnvVarName)
	if actID == "" {
		logging.Debug("Not in an activated environment, so no need to reactivate")
		return nil
	}

	fname := path.Join(p.cfg.ConfigPath(), constants.UpdateHailFileName)
	// must happen last in this function scope (defer if needed)
	if err := hail.Send(fname, []byte(actID)); err != nil {
		logging.Error("failed to send hail via %q: %s", fname, err)
		return locale.WrapError(err, "err_pull_hail", "Could not re-activate your project, please exit and re-activate manually by running 'state activate' again.")
	}

	return nil
}

func targetProject(prj *project.Project, overwrite string) (*project.Namespaced, error) {
	ns := prj.Namespace()
	if overwrite != "" {
		var err error
		ns, err = project.ParseNamespace(overwrite)
		if err != nil {
			return nil, locale.WrapInputError(err, "pull_set_project_parse_err", "Failed to parse namespace {{.V0}}", overwrite)
		}
	}

	// Retrieve commit ID to set the project to (if unset)
	if ns.CommitID == nil || *ns.CommitID == "" || prj.BranchName() != "" {
		var err error
		ns.CommitID, err = model.LatestCommitID(ns.Owner, ns.Project, prj.BranchName())
		if err != nil {
			return nil, locale.WrapError(err, "err_pull_commit", "Could not retrieve the latest commit for your project.")
		}
	}

	return ns, nil
}

func areCommitsRelated(targetCommit strfmt.UUID, sourceCommmit strfmt.UUID) (bool, error) {
	history, err := model.CommitHistoryFromID(targetCommit)
	if err != nil {
		return false, locale.WrapError(err, "err_pull_commit_history", "Could not fetch commit history for target project.")
	}

	for _, c := range history {
		if sourceCommmit.String() == c.CommitID.String() {
			return true, nil
		}
	}
	return false, nil
}
