package reset

import (
	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/internal/runbits/runtime"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildscript"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
)

const local = "LOCAL"

type Params struct {
	Force    bool
	CommitID string
}

type Reset struct {
	out       output.Outputer
	auth      *authentication.Auth
	prompt    prompt.Prompter
	project   *project.Project
	analytics analytics.Dispatcher
	svcModel  *model.SvcModel
	cfg       *config.Instance
}

type primeable interface {
	primer.Outputer
	primer.Auther
	primer.Prompter
	primer.Projecter
	primer.Configurer
	primer.Analyticer
	primer.SvcModeler
}

func New(prime primeable) *Reset {
	return &Reset{
		prime.Output(),
		prime.Auth(),
		prime.Prompt(),
		prime.Project(),
		prime.Analytics(),
		prime.SvcModel(),
		prime.Config(),
	}
}

func (r *Reset) Run(params *Params) error {
	if r.project == nil {
		return rationalize.ErrNoProject
	}
	r.out.Notice(locale.Tr("operating_message", r.project.NamespaceString(), r.project.Dir()))

	var commitID strfmt.UUID
	switch {
	case params.CommitID == "":
		latestCommit, err := model.BranchCommitID(r.project.Owner(), r.project.Name(), r.project.BranchName())
		if err != nil {
			return locale.WrapError(err, "err_reset_latest_commit", "Could not get latest commit ID")
		}
		localCommitID, err := localcommit.Get(r.project.Dir())
		if err != nil && !errs.Matches(err, &localcommit.ErrInvalidCommitID{}) {
			return errs.Wrap(err, "Unable to get local commit")
		}
		if *latestCommit == localCommitID {
			return locale.NewInputError("err_reset_latest", "You are already on the latest commit")
		}
		commitID = *latestCommit

	case params.CommitID == local:
		localCommitID, err := localcommit.Get(r.project.Dir())
		if err != nil {
			return errs.Wrap(err, "Unable to get local commit")
		}
		commitID = localCommitID

	case !strfmt.IsUUID(params.CommitID):
		return locale.NewInputError("err_reset_invalid_commitid", "Invalid commit ID")

	default:
		commitID = strfmt.UUID(params.CommitID)

		history, err := model.CommitHistoryFromID(commitID, r.auth)
		if err != nil || len(history) == 0 {
			return locale.WrapInputError(err, "err_reset_commitid", "The given commit ID does not exist")
		}
	}

	localCommitID, err := localcommit.Get(r.project.Dir())
	if err != nil && !errs.Matches(err, &localcommit.ErrInvalidCommitID{}) {
		return errs.Wrap(err, "Unable to get local commit")
	}
	r.out.Notice(locale.Tl("reset_commit", "Your project will be reset to [ACTIONABLE]{{.V0}}[/RESET]\n", commitID.String()))
	if commitID != localCommitID {
		defaultChoice := params.Force || !r.out.Config().Interactive
		confirm, err := r.prompt.Confirm("", locale.Tl("reset_confim", "Resetting is destructive, you will lose any changes that were not pushed. Are you sure you want to do this?"), &defaultChoice)
		if err != nil {
			return locale.WrapError(err, "err_reset_confirm", "Could not confirm reset choice")
		}
		if !confirm {
			return locale.NewInputError("err_reset_aborted", "Reset aborted by user")
		}
	}

	err = localcommit.Set(r.project.Dir(), commitID.String())
	if err != nil {
		return errs.Wrap(err, "Unable to set local commit")
	}

	// Ensure the buildscript exists. Normally we should never do this, but reset is used for resetting from a corrupted
	// state, so it is appropriate.
	if r.cfg.GetBool(constants.OptinBuildscriptsConfig) {
		if err := buildscript.Initialize(r.project.Dir(), r.auth); err != nil {
			return errs.Wrap(err, "Unable to initialize buildscript")
		}
	}

	_, err = runtime.SolveAndUpdate(
		runtime.NewRequest(r.auth, r.analytics, r.project, &commitID, target.TriggerReset, r.svcModel, r.cfg, runtime.OptOrderChanged),
		r.out,
	)
	if err != nil {
		return locale.WrapError(err, "err_refresh_runtime")
	}

	r.out.Print(output.Prepare(
		locale.Tl("reset_success", "Successfully reset to commit: [NOTICE]{{.V0}}[/RESET]", commitID.String()),
		&struct {
			CommitID string `json:"commitID"`
		}{
			commitID.String(),
		},
	))

	return nil
}
