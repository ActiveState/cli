package uninstall

import (
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/internal/runbits/reqop_runbit"
	"github.com/ActiveState/cli/internal/runbits/runtime/trigger"
	"github.com/ActiveState/cli/internal/sliceutils"
	"github.com/ActiveState/cli/pkg/buildscript"
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
	primer.CheckoutInfoer
}

// Params tracks the info required for running Uninstall.
type Params struct {
	Packages captain.PackagesValue
}

type requirement struct {
	Requested *captain.PackageValue `json:"requested"`
	Resolved  types.Requirement     `json:"resolved"`

	// Remainder are for display purposes only
	Type model.NamespaceType `json:"type"`
}

type requirements []*requirement

func (r requirements) String() string {
	result := []string{}
	for _, req := range r {
		if req.Resolved.Namespace != "" {
			result = append(result, fmt.Sprintf("%s/%s", req.Resolved.Namespace, req.Requested.Name))
		} else {
			result = append(result, req.Requested.Name)
		}
	}
	return strings.Join(result, ", ")
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
	bp := bpModel.NewBuildPlannerModel(u.prime.Auth(), u.prime.SvcModel())

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
	localCommitID, err := u.prime.CheckoutInfo().CommitID()
	if err != nil {
		return errs.Wrap(err, "Unable to get commit ID")
	}
	oldCommit, err := bp.FetchCommit(localCommitID, pj.Owner(), pj.Name(), pj.BranchName(), nil)
	if err != nil {
		return errs.Wrap(err, "Failed to fetch old build result")
	}

	// Update buildscript
	script := oldCommit.BuildScript()
	reqs, err := u.resolveRequirements(script, params.Packages)
	if err != nil {
		return errs.Wrap(err, "Failed to resolve requirements")
	}
	for _, req := range reqs {
		if err := script.RemoveRequirement(req.Resolved); err != nil {
			return errs.Wrap(err, "Unable to remove requirement")
		}
	}

	// Done updating requirements
	pg.Stop(locale.T("progress_success"))
	pg = nil

	// Update local checkout and source runtime changes
	if err := reqop_runbit.UpdateAndReload(u.prime, script, oldCommit, locale.Tr("commit_message_added", params.Packages.String()), trigger.TriggerUninstall); err != nil {
		return errs.Wrap(err, "Failed to update local checkout")
	}

	if out.Type().IsStructured() {
		out.Print(output.Structured(reqs))
	} else {
		u.renderUserFacing(reqs)
	}

	// All done
	out.Notice(locale.T("operation_success_local"))

	return nil
}

func (u *Uninstall) renderUserFacing(reqs requirements) {
	u.prime.Output().Notice("")
	for _, req := range reqs {
		l := "install_report_removed"
		u.prime.Output().Notice(locale.Tr(l, fmt.Sprintf("%s/%s", req.Resolved.Namespace, req.Resolved.Name)))
	}
	u.prime.Output().Notice("")
}

func (u *Uninstall) resolveRequirements(script *buildscript.BuildScript, pkgs captain.PackagesValue) (requirements, error) {
	result := requirements{}

	reqs, err := script.DependencyRequirements()
	if err != nil {
		return nil, errs.Wrap(err, "Unable to get requirements")
	}

	// Resolve requirements and check for errors
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

		// Check for duplicate matches
		if len(matches) > 1 {
			multipleMatches = append(multipleMatches, pkg)
			continue
		}

		// Check for no matches
		if len(matches) == 0 {
			notFound = append(notFound, pkg)
			continue
		}

		result = append(result, &requirement{
			Requested: pkg,
			Resolved:  matches[0],
			Type:      model.ParseNamespace(matches[0].Namespace).Type(),
		})
	}

	// Error out on duplicate matches
	if len(multipleMatches) > 0 {
		return result, &errMultipleMatches{error: errs.New("Could not find all requested packages"), packages: multipleMatches}
	}

	// Error out on no matches
	if len(notFound) > 0 {
		return result, &errNoMatches{error: errs.New("Could not find all requested packages"), packages: notFound}
	}

	return result, nil
}
