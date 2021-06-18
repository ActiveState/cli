package pull

import (
	"errors"
	"strings"

	"github.com/ActiveState/cli/internal/machineid"
	"github.com/ActiveState/cli/pkg/cmdlets/commit"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

type Pull struct {
	prompt  prompt.Prompter
	project *project.Project
	auth    *authentication.Auth
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
	primer.Auther
	primer.Outputer
	primer.Configurer
}

func New(prime primeable) *Pull {
	return &Pull{
		prime.Prompt(),
		prime.Project(),
		prime.Auth(),
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

	if !p.project.IsHeadless() && p.project.BranchName() == "" {
		return locale.NewError("err_pull_branch", "Your [NOTICE]activestate.yaml[/RESET] project field does not contain a branch. Please ensure you are using the latest version of the State Tool by running [ACTIONABLE]`state update`[/RESET] and then trying again.")
	}

	// Determine the project to pull from
	remoteProject, err := resolveRemoteProject(p.project, params.SetProject)
	if err != nil {
		return errs.Wrap(err, "Unable to determine target project")
	}

	var localCommit *strfmt.UUID
	if p.project.CommitUUID() != "" {
		v := p.project.CommitUUID()
		localCommit = &v
	}

	if params.SetProject != "" {
		if !params.Force {
			confirmed, err := p.prompt.Confirm(
				locale.T("confirm"),
				locale.Tl("confirm_unrelated_pull_set_project",
					"If you switch to {{.V0}}, you may lose changes to your project. Are you sure you want to do this?", remoteProject.String()),
				new(bool))
			if err != nil {
				return locale.WrapError(err, "err_pull_confirm", "Failed to get user confirmation to update project")
			}
			if !confirmed {
				return locale.NewInputError("err_pull_aborted", "Pull aborted by user")
			}
		}

		err = p.project.Source().SetNamespace(remoteProject.Owner, remoteProject.Project)
		if err != nil {
			return locale.WrapError(err, "err_pull_update_namespace", "Cannot update the namespace in your project file.")
		}
	}

	remoteCommit := remoteProject.CommitID
	resultingCommit := remoteCommit // resultingCommit is the commit we want to update the local project file with

	if localCommit != nil {
		strategies, err := model.MergeCommit(*remoteCommit, *localCommit)
		if err != nil {
			if errors.Is(err, model.ErrMergeFastForward) {
				// No merge necessary
				resultingCommit = localCommit
			} else if !errors.Is(err, model.ErrMergeCommitInHistory) {
				return locale.WrapError(err, "err_mergecommit", "Could not detect if merge is necessary.")
			}
		}
		if err == nil && strategies != nil {
			c, err := p.performMerge(strategies, *remoteCommit)
			if err != nil {
				return errs.Wrap(err, "performing merge commit failed")
			}
			resultingCommit = &c
		}
	}

	// Update the commit ID in the activestate.yaml
	if p.project.CommitID() != resultingCommit.String() {
		err := p.project.Source().SetCommit(resultingCommit.String(), false)
		if err != nil {
			return locale.WrapError(err, "err_pull_update", "Cannot update the commit in your project file.")
		}

		p.out.Print(&outputFormat{
			locale.Tr("pull_updated", remoteProject.String(), resultingCommit.String()),
			true,
		})
	} else {
		p.out.Print(&outputFormat{
			locale.Tl("pull_not_updated", "Your activestate.yaml is already up to date."),
			false,
		})
	}

	err = runbits.RefreshRuntime(p.auth, p.out, p.project, p.cfg.CachePath(), *resultingCommit, true)
	if err != nil {
		return locale.WrapError(err, "err_pull_refresh", "Could not refresh runtime after pull")
	}

	return nil
}

func (p *Pull) performMerge(strategies *mono_models.MergeStrategies, remoteCommit strfmt.UUID) (strfmt.UUID, error) {
	p.out.Notice(output.Heading(locale.Tl("pull_diverged", "Merging history")))
	p.out.Notice(locale.Tr(
		"pull_diverged_message",
		p.project.Namespace().String(), p.project.BranchName(), p.project.CommitID(), remoteCommit.String()))

	commitMessage := locale.Tr("pull_merge_commit", remoteCommit.String(), p.project.CommitID())
	resultCommit, err := model.CommitChangeset(remoteCommit, commitMessage, machineid.UniqID(), strategies.OverwriteChanges)
	if err != nil {
		return "", locale.WrapError(err, "err_pull_merge_commit", "Could not create merge commit.")
	}

	cmit, err := model.GetCommit(resultCommit)
	if err != nil {
		return "", locale.WrapError(err, "err_pull_getcommit", "Could not inspect resulting commit.")
	}
	p.out.Notice(locale.Tl(
		"pull_diverged_changes",
		"The following changes will be merged:\n{{.V0}}\n", strings.Join(commit.FormatChanges(cmit), "\n")))

	return resultCommit, nil
}

func resolveRemoteProject(prj *project.Project, overwrite string) (*project.Namespaced, error) {
	ns := prj.Namespace()
	if overwrite != "" {
		var err error
		ns, err = project.ParseNamespace(overwrite)
		if err != nil {
			return nil, locale.WrapInputError(err, "pull_set_project_parse_err", "Failed to parse namespace {{.V0}}", overwrite)
		}
	}

	// Retrieve commit ID to set the project to (if unset)
	if overwrite != "" {
		branch, err := model.DefaultBranchForProjectName(ns.Owner, ns.Project)
		if err != nil {
			return nil, locale.WrapError(err, "err_pull_commit", "Could not retrieve the latest commit for your project.")
		}
		ns.CommitID = branch.CommitID
	} else {
		var err error
		ns.CommitID, err = model.BranchCommitID(ns.Owner, ns.Project, prj.BranchName())
		if err != nil {
			return nil, locale.WrapError(err, "err_pull_commit_branch", "Could not retrieve the latest commit for your project and branch.")
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
