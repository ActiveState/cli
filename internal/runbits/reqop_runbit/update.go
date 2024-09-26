package reqop_runbit

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits/buildscript"
	"github.com/ActiveState/cli/internal/runbits/cves"
	"github.com/ActiveState/cli/internal/runbits/dependencies"
	"github.com/ActiveState/cli/internal/runbits/runtime"
	"github.com/ActiveState/cli/internal/runbits/runtime/trigger"
	"github.com/ActiveState/cli/pkg/buildscript"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/model/buildplanner"
	"github.com/ActiveState/cli/pkg/runtime"
	"github.com/go-openapi/strfmt"
)

type primeable interface {
	primer.Outputer
	primer.Prompter
	primer.Projecter
	primer.Auther
	primer.Configurer
	primer.Analyticer
	primer.SvcModeler
	primer.CheckoutInfoer
}

type requirements []*Requirement

func (r requirements) String() string {
	result := []string{}
	for _, req := range r {
		result = append(result, fmt.Sprintf("%s/%s", req.Namespace, req.Name))
	}
	return strings.Join(result, ", ")
}

type Requirement struct {
	Name      string
	Namespace model.Namespace
	Version   []types.VersionRequirement
}

func UpdateAndReload(prime primeable, script *buildscript.BuildScript, oldCommit *buildplanner.Commit, commitMsg string, trigger trigger.Trigger) error {
	pj := prime.Project()
	out := prime.Output()
	cfg := prime.Config()
	bp := buildplanner.NewBuildPlannerModel(prime.Auth(), prime.SvcModel())

	var pg *output.Spinner
	defer func() {
		if pg != nil {
			pg.Stop(locale.T("progress_fail"))
		}
	}()
	pg = output.StartSpinner(out, locale.T("progress_solve_preruntime"), constants.TerminalAnimationInterval)

	commitParams := buildplanner.StageCommitParams{
		Owner:        pj.Owner(),
		Project:      pj.Name(),
		ParentCommit: string(oldCommit.CommitID),
		Description:  commitMsg,
		Script:       script,
	}

	// Solve runtime
	newCommit, err := bp.StageCommit(commitParams)
	if err != nil {
		return errs.Wrap(err, "Could not stage commit")
	}

	// Stop process of creating the commit
	pg.Stop(locale.T("progress_success"))
	pg = nil

	// Report changes and CVEs to user
	dependencies.OutputChangeSummary(out, newCommit.BuildPlan(), oldCommit.BuildPlan())
	if err := cves.NewCveReport(prime).Report(newCommit.BuildPlan(), oldCommit.BuildPlan()); err != nil {
		return errs.Wrap(err, "Could not report CVEs")
	}

	// Start runtime sourcing UI
	// Note normally we'd defer to Update's logic of async runtimes, but the reason we do this is to allow for solve
	// errors to still be relayed even when using async. In this particular case the solving logic already happened
	// when we created the commit, so running it again doesn't provide any value and only would slow things down.
	if !cfg.GetBool(constants.AsyncRuntimeConfig) {
		// refresh or install runtime
		_, err := runtime_runbit.Update(prime, trigger,
			runtime_runbit.WithCommit(newCommit),
			runtime_runbit.WithoutBuildscriptValidation(),
		)
		if err != nil {
			if !isBuildError(err) {
				// If the error is not a build error we still want to update the commit
				if err2 := updateCommitID(prime, newCommit.CommitID); err2 != nil {
					return errs.Pack(err, locale.WrapError(err2, "err_package_update_commit_id"))
				}
			}
			return errs.Wrap(err, "Failed to refresh runtime")
		}
	} else {
		prime.Output().Notice("") // blank line
		prime.Output().Notice(locale.Tr("notice_async_runtime", constants.AsyncRuntimeConfig))
	}

	// Update commit ID
	if err := updateCommitID(prime, newCommit.CommitID); err != nil {
		return locale.WrapError(err, "err_package_update_commit_id")
	}
	return nil
}

func updateCommitID(prime primeable, commitID strfmt.UUID) error {
	if err := localcommit.Set(prime.Project().Dir(), commitID.String()); err != nil {
		return locale.WrapError(err, "err_package_update_commit_id")
	}

	if prime.Config().GetBool(constants.OptinBuildscriptsConfig) {
		bp := buildplanner.NewBuildPlannerModel(prime.Auth(), prime.SvcModel())
		script, err := bp.GetBuildScript(commitID.String())
		if err != nil {
			return errs.Wrap(err, "Could not get remote build expr and time")
		}

		err = buildscript_runbit.Update(prime.Project(), script)
		if err != nil {
			return locale.WrapError(err, "err_update_build_script")
		}
	}

	return nil
}

func isBuildError(err error) bool {
	var errBuild *runtime.BuildError
	var errBuildPlanner *response.BuildPlannerError

	return errors.As(err, &errBuild) || errors.As(err, &errBuildPlanner)
}
