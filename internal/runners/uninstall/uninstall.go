package uninstall

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/internal/runbits/reqop_runbit"
	"github.com/ActiveState/cli/internal/sliceutils"
	"github.com/ActiveState/cli/pkg/buildscript"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	"github.com/ActiveState/cli/pkg/platform/model"
	bpModel "github.com/ActiveState/cli/pkg/platform/model/buildplanner"
)

type primeable interface {
	primer.Outputer
	primer.Prompter
	primer.Projecter
	primer.Auther
	primer.Configurer
	primer.Analyticer
	primer.SvcModeler
}

// Params tracks the info required for running Uninstall.
type Params struct {
	Packages captain.PackagesValue
}

// Uninstall manages the installing execution context.
type Uninstall struct {
	prime  primeable
	nsType model.NamespaceType
}

// New prepares an installation execution context for use.
func New(prime primeable, nsType model.NamespaceType) *Uninstall {
	return &Uninstall{prime, nsType}
}

type errNoMatches struct {
	error
	packages captain.PackagesValue
}

type errMultipleMatches struct {
	error
	packages captain.PackagesValue
}

// Run executes the install behavior.
func (u *Uninstall) Run(params Params) (rerr error) {
	defer u.rationalizeError(&rerr)

	logging.Debug("ExecuteUninstall")

	pj := u.prime.Project()
	out := u.prime.Output()
	bp := bpModel.NewBuildPlannerModel(u.prime.Auth())

	// Verify input
	if pj == nil {
		return rationalize.ErrNoProject
	}
	if pj.IsHeadless() {
		return rationalize.ErrHeadless
	}

	out.Notice(locale.Tr("operating_message", pj.NamespaceString(), pj.Dir()))

	var pg *output.Spinner
	defer func() {
		if pg != nil {
			pg.Stop(locale.T("progress_fail"))
		}
	}()

	// Start process of updating requirements
	pg = output.StartSpinner(out, locale.T("progress_requirements"), constants.TerminalAnimationInterval)

	// Grab local commit info
	localCommitID, err := localcommit.Get(u.prime.Project().Dir())
	if err != nil {
		return errs.Wrap(err, "Unable to get local commit")
	}
	oldCommit, err := bp.FetchCommit(localCommitID, pj.Owner(), pj.Name(), nil)
	if err != nil {
		return errs.Wrap(err, "Failed to fetch old build result")
	}

	// Update buildscript
	script := oldCommit.BuildScript()
	if err := u.prepareBuildScript(script, params.Packages); err != nil {
		return errs.Wrap(err, "Could not prepare build script")
	}

	// Done updating requirements
	pg.Stop(locale.T("progress_success"))
	pg = nil

	// Update local checkout and source runtime changes
	if err := reqop_runbit.UpdateAndReload(u.prime, script, oldCommit, locale.Tr("commit_message_added", params.Packages.String())); err != nil {
		return errs.Wrap(err, "Failed to update local checkout")
	}

	// All done
	out.Notice(locale.T("operation_success_local"))

	return nil
}

func (u *Uninstall) prepareBuildScript(script *buildscript.BuildScript, pkgs captain.PackagesValue) error {
	reqs, err := script.DependencyRequirements()
	if err != nil {
		return errs.Wrap(err, "Unable to get requirements")
	}

	// Resolve requirements and check for errors
	toRemove := []types.Requirement{}
	notFound := captain.PackagesValue{}
	multipleMatches := captain.PackagesValue{}
	for _, pkg := range pkgs {
		// Filter matching requirements
		matches := sliceutils.Filter(reqs, func(req types.Requirement) bool {
			if pkg.Name != req.Name {
				return false
			}
			if pkg.Namespace != "" {
				return req.Namespace == pkg.Namespace
			}
			return model.NamespaceMatch(req.Namespace, u.nsType.Matchable())
		})
		toRemove = append(toRemove, matches...)

		// Check for duplicate matches
		if len(matches) > 1 {
			multipleMatches = append(multipleMatches, pkg)
		}

		// Check for no matches
		if len(matches) == 0 {
			notFound = append(notFound, pkg)
		}
	}

	// Error out on duplicate matches
	if len(multipleMatches) > 0 {
		return &errMultipleMatches{error: errs.New("Could not find all requested packages"), packages: multipleMatches}
	}

	// Error out on no matches
	if len(notFound) > 0 {
		return &errNoMatches{error: errs.New("Could not find all requested packages"), packages: notFound}
	}

	// Remove requirements
	for _, req := range toRemove {
		if err := script.RemoveRequirement(req); err != nil {
			return errs.Wrap(err, "Unable to remove requirement")
		}
	}

	return nil
}
