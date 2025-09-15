package pull

import (
	"errors"
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
	"github.com/ActiveState/cli/internal/runbits/buildscript"
	"github.com/ActiveState/cli/internal/runbits/commit"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/internal/runbits/runtime"
	"github.com/ActiveState/cli/internal/runbits/runtime/trigger"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/model/buildplanner"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
)

type Pull struct {
	prime primeable
	// The remainder is redundant with the above. Refactoring this will follow in a later story so as not to blow
	// up the one that necessitates adding the primer at this level.
	// https://activestatef.atlassian.net/browse/DX-2869
	prompt    prompt.Prompter
	project   *project.Project
	auth      *authentication.Auth
	out       output.Outputer
	analytics analytics.Dispatcher
	cfg       *config.Instance
	svcModel  *model.SvcModel
}

type errNoCommonParent struct {
	localCommitID  strfmt.UUID
	remoteCommitID strfmt.UUID
}

func (e errNoCommonParent) Error() string {
	return "no common parent"
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
		prime,
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

type ErrBuildScriptMergeConflict struct {
	ProjectDir string
}

func (e *ErrBuildScriptMergeConflict) Error() string {
	return "build script merge conflict"
}

func (p *Pull) Run(params *PullParams) (rerr error) {
	defer rationalizeError(&rerr)

	if p.project == nil {
		return rationalize.ErrNoProject
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
		commonParent, err := model.CommonParent(localCommit, remoteCommit, p.auth)
		if err != nil {
			return errs.Wrap(err, "Unable to determine common parent")
		}

		if commonParent == nil {
			return &errNoCommonParent{
				*localCommit,
				*remoteCommit,
			}
		}

		// Attempt to fast-forward merge. This will succeed if the commits are
		// compatible, meaning that we can simply update the local commit ID to
		// the remoteCommit ID. The commitID returned from MergeCommit with this
		// strategy should just be the remote commit ID.
		// If this call fails then we will try a recursive merge.
		strategy := types.MergeCommitStrategyFastForward

		bp := buildplanner.NewBuildPlannerModel(p.auth, p.svcModel)
		params := &buildplanner.MergeCommitParams{
			Owner:     remoteProject.Owner,
			Project:   remoteProject.Project,
			TargetRef: localCommit.String(),
			OtherRef:  remoteCommit.String(),
			Strategy:  strategy,
		}

		resultCommit, mergeErr := bp.MergeCommit(params)
		if mergeErr != nil {
			logging.Debug("Merge with fast-forward failed with error: %s, trying recursive overwrite", mergeErr.Error())
			strategy = types.MergeCommitStrategyRecursiveKeepOnConflict
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
		if p.cfg.GetBool(constants.OptinBuildscriptsConfig) {
			err := p.mergeBuildScript(*remoteCommit, *localCommit)
			if err != nil {
				var errBuildScriptMergeConflict *ErrBuildScriptMergeConflict
				if errors.As(err, &errBuildScriptMergeConflict) {
					err2 := localcommit.Set(p.project.Dir(), remoteCommit.String())
					if err2 != nil {
						err = errs.Pack(err, errs.Wrap(err2, "Could not set local commit to remote commit after build script merge conflict"))
					}
				}
				return errs.Wrap(err, "Could not merge local build script with remote changes")
			}
		}

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

	_, err = runtime_runbit.Update(p.prime, trigger.TriggerPull)
	if err != nil {
		return locale.WrapError(err, "err_pull_refresh", "Could not refresh runtime after pull")
	}

	return nil
}

func (p *Pull) performMerge(remoteCommit, localCommit strfmt.UUID, namespace *project.Namespaced, branchName string, strategy types.MergeStrategy) (strfmt.UUID, error) {
	p.out.Notice(output.Title(locale.Tl("pull_diverged", "Merging history")))
	p.out.Notice(locale.Tr(
		"pull_diverged_message",
		namespace.String(), branchName, localCommit.String(), remoteCommit.String()),
	)

	bp := buildplanner.NewBuildPlannerModel(p.auth, p.svcModel)
	params := &buildplanner.MergeCommitParams{
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

	cmit, err := model.GetCommit(resultCommit, p.auth)
	if err != nil {
		return "", locale.WrapError(err, "err_pull_getcommit", "Could not inspect resulting commit.")
	}
	if changes, _ := commit.FormatChanges(cmit); len(changes) > 0 {
		p.out.Notice(locale.Tl(
			"pull_diverged_changes",
			"The following changes will be merged:\n{{.V0}}\n", strings.Join(changes, "\n")),
		)
	}

	return resultCommit, nil
}

// mergeBuildScript merges the local build script with the remote buildscript.
func (p *Pull) mergeBuildScript(remoteCommit, localCommit strfmt.UUID) error {
	// Get the build script to merge.
	scriptA, err := buildscript_runbit.ScriptFromProject(p.project)
	if err != nil {
		return errs.Wrap(err, "Could not get local build script")
	}

	// Get the local and remote build expressions to merge.
	bp := buildplanner.NewBuildPlannerModel(p.auth, p.svcModel)
	scriptB, err := bp.GetBuildScript(remoteCommit.String())
	if err != nil {
		return errs.Wrap(err, "Unable to get buildexpression and time for remote commit")
	}

	// Compute the merge strategy.
	strategies, err := model.MergeCommit(remoteCommit, localCommit)
	if err != nil {
		if errors.Is(err, model.ErrMergeFastForward) || errors.Is(err, model.ErrMergeCommitInHistory) {
			return buildscript_runbit.Update(p.project, scriptB)
		}
		return locale.WrapError(err, "err_mergecommit", "Could not detect if merge is necessary.")
	}

	// Attempt the merge.
	err = scriptA.Merge(scriptB, strategies)
	if err != nil {
		err := buildscript_runbit.GenerateAndWriteDiff(p.project, scriptA, scriptB)
		if err != nil {
			return locale.WrapError(err, "err_diff_build_script", "Unable to generate differences between local and remote build script")
		}
		return &ErrBuildScriptMergeConflict{p.project.Dir()}
	}

	// Write the merged build expression as a local build script.
	return buildscript_runbit.Update(p.project, scriptA)
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
