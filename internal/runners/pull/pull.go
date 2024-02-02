package pull

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/analytics"
	anaConst "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/analytics/dimensions"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/runbits"
	buildscriptRunbits "github.com/ActiveState/cli/internal/runbits/buildscript"
	"github.com/ActiveState/cli/internal/runbits/commit"
	"github.com/ActiveState/cli/pkg/localcommit"
	bpModel "github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildexpression/merge"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildscript"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
)

type Pull struct {
	prompt    prompt.Prompter
	project   *project.Project
	auth      *authentication.Auth
	out       output.Outputer
	analytics analytics.Dispatcher
	cfg       *config.Instance
	svcModel  *model.SvcModel
}

type PullParams struct {
	Force bool
}

type primeable interface {
	primer.Prompter
	primer.Projecter
	primer.Auther
	primer.Outputer
	primer.Analyticer
	primer.Configurer
	primer.SvcModeler
}

func New(prime primeable) *Pull {
	return &Pull{
		prime.Prompt(),
		prime.Project(),
		prime.Auth(),
		prime.Output(),
		prime.Analytics(),
		prime.Config(),
		prime.SvcModel(),
	}
}

type pullOutput struct {
	Message string `locale:"message,Message" json:"message"`
	Success bool   `locale:"success,Success" json:"success"`
}

func (o *pullOutput) MarshalOutput(format output.Format) interface{} {
	return o.Message
}

func (o *pullOutput) MarshalStructured(format output.Format) interface{} {
	return o
}

func (p *Pull) Run(params *PullParams) (rerr error) {
	defer rationalizeError(&rerr)

	if p.project == nil {
		return locale.NewInputError("err_no_project")
	}
	p.out.Notice(locale.Tr("operating_message", p.project.NamespaceString(), p.project.Dir()))

	if p.project.IsHeadless() {
		return locale.NewInputError("err_pull_headless", "You must first create a project. Please visit {{.V0}} to create your project.", p.project.URL())
	}

	if p.project.BranchName() == "" {
		return locale.NewError("err_pull_branch", "Your [NOTICE]activestate.yaml[/RESET] project field does not contain a branch. Please ensure you are using the latest version of the State Tool by running '[ACTIONABLE]state update[/RESET]' and then trying again.")
	}

	// Determine the project to pull from
	remoteProject, err := resolveRemoteProject(p.project)
	if err != nil {
		return errs.Wrap(err, "Unable to determine target project")
	}

	var localCommit *strfmt.UUID
	localCommitID, err := localcommit.Get(p.project.Dir())
	if err != nil {
		return errs.Wrap(err, "Unable to get local commit")
	}
	if localCommitID != "" {
		localCommit = &localCommitID
	}

	remoteCommit := remoteProject.CommitID
	resultingCommit := remoteCommit // resultingCommit is the commit we want to update the local project file with

	if localCommit != nil {
		// Attempt to fast-forward merge. This will succeed if the commits are
		// compatible, meaning that we can simply update the local commit ID to
		// the remoteCommit ID. The commitID returned from MergeCommit with this
		// strategy should just be the remote commit ID.
		// If this call fails then we will try a recursive merge.
		strategy := bpModel.MergeCommitStrategyFastForward

		bp := model.NewBuildPlannerModel(p.auth)
		params := &model.MergeCommitParams{
			Owner:     remoteProject.Owner,
			Project:   remoteProject.Project,
			TargetRef: localCommit.String(),
			OtherRef:  remoteCommit.String(),
			Strategy:  strategy,
		}

		resultCommit, mergeErr := bp.MergeCommit(params)
		if mergeErr != nil {
			logging.Debug("Merge with fast-forward failed with error: %s, trying recursive overwrite", mergeErr.Error())
			strategy = bpModel.MergeCommitStrategyRecursiveOverwriteOnConflict
			c, err := p.performMerge(*remoteCommit, *localCommit, remoteProject, p.project.BranchName(), strategy)
			if err != nil {
				p.notifyMergeStrategy(anaConst.LabelVcsConflictMergeStrategyFailed, *localCommit, remoteProject)
				return errs.Wrap(err, "performing merge commit failed")
			}
			resultingCommit = &c
		} else {
			logging.Debug("Fast-forward merge succeeded, setting commit ID to %s", resultCommit.String())
			resultingCommit = &resultCommit
		}

		p.notifyMergeStrategy(string(strategy), *localCommit, remoteProject)
	}

	commitID, err := localcommit.Get(p.project.Dir())
	if err != nil {
		return errs.Wrap(err, "Unable to get local commit")
	}

	if commitID != *resultingCommit {
		err := localcommit.Set(p.project.Dir(), resultingCommit.String())
		if err != nil {
			return errs.Wrap(err, "Unable to set local commit")
		}

		p.out.Print(&pullOutput{
			locale.Tr("pull_updated", remoteProject.String(), resultingCommit.String()),
			true,
		})
	} else {
		p.out.Print(&pullOutput{
			locale.Tl("pull_not_updated", "Your project is already up to date."),
			false,
		})
	}

	err = runbits.RefreshRuntime(p.auth, p.out, p.analytics, p.project, *resultingCommit, true, target.TriggerPull, p.svcModel, p.cfg)
	if err != nil {
		return locale.WrapError(err, "err_pull_refresh", "Could not refresh runtime after pull")
	}

	return nil
}

func (p *Pull) performMerge(remoteCommit, localCommit strfmt.UUID, namespace *project.Namespaced, branchName string, strategy bpModel.MergeStrategy) (strfmt.UUID, error) {
	err := p.mergeBuildScript(remoteCommit, localCommit)
	if err != nil {
		return "", errs.Wrap(err, "Could not merge local build script with remote changes")
	}

	p.out.Notice(output.Title(locale.Tl("pull_diverged", "Merging history")))
	p.out.Notice(locale.Tr(
		"pull_diverged_message",
		namespace.String(), branchName, localCommit.String(), remoteCommit.String()),
	)

	bp := model.NewBuildPlannerModel(p.auth)
	params := &model.MergeCommitParams{
		Owner:     namespace.Owner,
		Project:   namespace.Project,
		TargetRef: localCommit.String(),
		OtherRef:  remoteCommit.String(),
		Strategy:  strategy,
	}
	resultCommit, err := bp.MergeCommit(params)
	if err != nil {
		return "", locale.WrapError(err, "err_pull_merge_commit", "Could not create merge commit.")
	}

	cmit, err := model.GetCommit(resultCommit)
	if err != nil {
		return "", locale.WrapError(err, "err_pull_getcommit", "Could not inspect resulting commit.")
	}
	changes, _ := commit.FormatChanges(cmit)
	p.out.Notice(locale.Tl(
		"pull_diverged_changes",
		"The following changes will be merged:\n{{.V0}}\n", strings.Join(changes, "\n")),
	)

	return resultCommit, nil
}

// mergeBuildScript merges the local build script with the remote buildexpression (not script).
func (p *Pull) mergeBuildScript(remoteCommit, localCommit strfmt.UUID) error {
	// Get the build script to merge.
	script, err := buildscript.NewScriptFromProject(p.project, p.auth)
	if err != nil {
		return errs.Wrap(err, "Could not get local build script")
	}

	// Get the local and remote build expressions to merge.
	exprA := script.Expr
	bp := model.NewBuildPlannerModel(p.auth)
	exprB, err := bp.GetBuildExpression(p.project.Owner(), p.project.Name(), remoteCommit.String())
	if err != nil {
		return errs.Wrap(err, "Unable to get buildexpression for remote commit")
	}

	// Compute the merge strategy.
	strategies, err := model.MergeCommit(remoteCommit, localCommit)
	if err != nil {
		if !errors.Is(err, model.ErrMergeCommitInHistory) {
			return locale.WrapError(err, "err_mergecommit", "Could not detect if merge is necessary.")
		}
	}

	// Attempt the merge.
	mergedExpr, err := merge.Merge(exprA, exprB, strategies)
	if err != nil {
		err := buildscriptRunbits.GenerateAndWriteDiff(p.project, script, exprB)
		if err != nil {
			return locale.WrapError(err, "err_diff_build_script", "Unable to generate differences between local and remote build script")
		}
		return locale.NewInputError(
			"err_build_script_merge",
			"Unable to automatically merge build scripts. Please resolve conflicts manually in '{{.V0}}' and then run '[ACTIONABLE]state commit[/RESET]'",
			filepath.Join(p.project.Dir(), constants.BuildScriptFileName))
	}

	// Write the merged build expression as a local build script.
	return buildscript.Update(p.project, mergedExpr, p.auth)
}

func resolveRemoteProject(prj *project.Project) (*project.Namespaced, error) {
	ns := prj.Namespace()
	var err error
	ns.CommitID, err = model.BranchCommitID(ns.Owner, ns.Project, prj.BranchName())
	if err != nil {
		return nil, locale.WrapError(err, "err_pull_commit_branch", "Could not retrieve the latest commit for your project and branch.")
	}

	return ns, nil
}

func (p *Pull) notifyMergeStrategy(strategy string, commitID strfmt.UUID, namespace *project.Namespaced) {
	p.analytics.EventWithLabel(anaConst.CatInteractions, anaConst.ActVcsConflict, strategy, &dimensions.Values{
		CommitID:         ptr.To(commitID.String()),
		ProjectNameSpace: ptr.To(namespace.String()),
	})
}
