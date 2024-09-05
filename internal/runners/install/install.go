package install

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	buildscript_runbit "github.com/ActiveState/cli/internal/runbits/buildscript"
	"github.com/ActiveState/cli/internal/runbits/commits_runbit"
	"github.com/ActiveState/cli/internal/runbits/cves"
	"github.com/ActiveState/cli/internal/runbits/dependencies"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	runtime_runbit "github.com/ActiveState/cli/internal/runbits/runtime"
	"github.com/ActiveState/cli/internal/runbits/runtime/trigger"
	"github.com/ActiveState/cli/internal/sliceutils"
	"github.com/ActiveState/cli/pkg/buildscript"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	"github.com/ActiveState/cli/pkg/platform/model"
	bpModel "github.com/ActiveState/cli/pkg/platform/model/buildplanner"
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
}

// InstallRunParams tracks the info required for running Install.
type InstallRunParams struct {
	Packages  captain.PackagesValue
	Timestamp captain.TimeValue
}

type requirement struct {
	input              *captain.PackageValue
	resolvedVersionReq []types.VersionRequirement
	resolvedNamespace  *model.Namespace
	matchedIngredients []*model.IngredientAndVersion
}

type requirements []*requirement

func (r requirements) String() string {
	result := []string{}
	for _, req := range r {
		if req.resolvedNamespace != nil {
			result = append(result, fmt.Sprintf("%s/%s", req.resolvedNamespace.String(), req.input.Name))
		} else {
			result = append(result, req.input.Name)
		}
	}
	return strings.Join(result, ", ")
}

// Install manages the installing execution context.
type Install struct {
	prime primeable
}

// NewInstall prepares an installation execution context for use.
func NewInstall(prime primeable) *Install {
	return &Install{prime}
}

// Run executes the install behavior.
func (i *Install) Run(params InstallRunParams) (rerr error) {
	defer rationalizeError(i.prime.Auth(), &rerr)

	logging.Debug("ExecuteInstall")

	pj := i.prime.Project()
	out := i.prime.Output()
	bp := bpModel.NewBuildPlannerModel(i.prime.Auth())

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
			// This is a bit awkward, but it would be even more awkward to manually address this for every error condition
			pg.Stop(locale.T("progress_fail"))
		}
	}()

	// Start process of resolving requirements
	var err error
	var oldCommit *bpModel.Commit
	var reqs requirements
	var ts time.Time
	{
		pg = output.StartSpinner(out, locale.T("progress_search"), constants.TerminalAnimationInterval)

		// Resolve timestamp, commit and languages used for current project.
		// This will be used to resolve the requirements.
		ts, err = commits_runbit.ExpandTimeForProject(&params.Timestamp, i.prime.Auth(), i.prime.Project())
		if err != nil {
			return errs.Wrap(err, "Unable to get timestamp from params")
		}

		// Grab local commit info
		localCommitID, err := localcommit.Get(i.prime.Project().Dir())
		if err != nil {
			return errs.Wrap(err, "Unable to get local commit")
		}
		oldCommit, err = bp.FetchCommit(localCommitID, pj.Owner(), pj.Name(), nil)
		if err != nil {
			return errs.Wrap(err, "Failed to fetch old build result")
		}

		// Get languages used in current project
		languages, err := model.FetchLanguagesForCommit(localCommitID, i.prime.Auth())
		if err != nil {
			logging.Debug("Could not get language from project: %v", err)
		}

		// Resolve requirements
		reqs, err = i.resolveRequirements(params.Packages, ts, languages)
		if err != nil {
			return errs.Wrap(err, "Unable to resolve requirements")
		}

		// Done resolving requirements
		pg.Stop(locale.T("progress_found"))
	}

	// Start process of creating the commit, which also solves it
	var newCommit *bpModel.Commit
	{
		pg = output.StartSpinner(out, locale.T("progress_solve_preruntime"), constants.TerminalAnimationInterval)

		script, err := i.prepareBuildScript(oldCommit.BuildScript(), reqs, ts)
		if err != nil {
			return errs.Wrap(err, "Could not prepare build script")
		}

		bsv, _ := script.Marshal()
		logging.Debug("Buildscript: %s", string(bsv))

		commitParams := bpModel.StageCommitParams{
			Owner:        pj.Owner(),
			Project:      pj.Name(),
			ParentCommit: string(oldCommit.CommitID),
			Description:  locale.Tr("commit_message_added", reqs.String()),
			Script:       script,
		}

		// Solve runtime
		newCommit, err = bp.StageCommit(commitParams)
		if err != nil {
			return errs.Wrap(err, "Could not stage commit")
		}

		// Stop process of creating the commit
		pg.Stop(locale.T("progress_success"))
		pg = nil
	}

	// Report changes and CVEs to user
	{
		dependencies.OutputChangeSummary(out, newCommit.BuildPlan(), oldCommit.BuildPlan())
		if err := cves.NewCveReport(i.prime).Report(newCommit.BuildPlan(), oldCommit.BuildPlan()); err != nil {
			return errs.Wrap(err, "Could not report CVEs")
		}
	}

	// Start runtime sourcing UI
	if !i.prime.Config().GetBool(constants.AsyncRuntimeConfig) {
		// refresh or install runtime
		_, err := runtime_runbit.Update(i.prime, trigger.TriggerInstall,
			runtime_runbit.WithCommit(newCommit),
			runtime_runbit.WithoutBuildscriptValidation(),
		)
		if err != nil {
			if !IsBuildError(err) {
				// If the error is not a build error we still want to update the commit
				if err2 := i.updateCommitID(newCommit.CommitID); err2 != nil {
					return errs.Pack(err, locale.WrapError(err2, "err_package_update_commit_id"))
				}
			}
			return errs.Wrap(err, "Failed to refresh runtime")
		}
	}

	// Update commit ID
	if err := i.updateCommitID(newCommit.CommitID); err != nil {
		return locale.WrapError(err, "err_package_update_commit_id")
	}

	// All done
	out.Notice(locale.T("operation_success_local"))

	return nil
}

type errNoMatches struct {
	error
	requirements []*requirement
}

// resolveRequirements will attempt to resolve the ingredient and namespace for each requested package
func (i *Install) resolveRequirements(packages captain.PackagesValue, ts time.Time, languages []model.Language) (requirements, error) {
	var disambiguate []*requirement
	var failed []*requirement
	reqs := []*requirement{}
	for _, pkg := range packages {
		req := &requirement{input: &pkg}
		if pkg.Namespace != "" {
			req.resolvedNamespace = ptr.To(model.NewNamespaceRaw(pkg.Namespace))
		}

		// Find ingredients that match the pkg query
		ingredients, err := model.SearchIngredientsStrict(pkg.Namespace, pkg.Name, false, false, &ts, i.prime.Auth())
		if err != nil {
			return nil, locale.WrapError(err, "err_pkgop_search_err", "Failed to check for ingredients.")
		}

		// Resolve matched ingredients
		if pkg.Namespace == "" {
			// Filter out ingredients that don't target one of the supported languages
			ingredients = sliceutils.Filter(ingredients, func(i *model.IngredientAndVersion) bool {
				il := model.LanguageFromNamespace(*i.Ingredient.PrimaryNamespace)
				for _, l := range languages {
					if l.Name == il {
						return true
					}
				}
				return false
			})
		}
		req.matchedIngredients = ingredients

		// Validate that the ingredient is resolved, and prompt the user if multiple ingredients matched
		if req.resolvedNamespace == nil {
			len := len(ingredients)
			switch {
			case len == 1:
				req.resolvedNamespace = ptr.To(model.ParseNamespace(*ingredients[0].Ingredient.PrimaryNamespace))
			case len > 1:
				disambiguate = append(disambiguate, req)
			case len == 0:
				failed = append(failed, req)
			}
		}

		reqs = append(reqs, req)
	}

	// Fail if not all requirements could be resolved
	if len(failed) > 0 {
		return nil, errNoMatches{error: errs.New("Failed to resolve requirements"), requirements: failed}
	}

	// Disambiguate requirements that match multiple ingredients
	if len(disambiguate) > 0 {
		for _, req := range disambiguate {
			ingredient, err := i.promptForMatchingIngredient(req)
			if err != nil {
				return nil, errs.Wrap(err, "Prompting for namespace failed")
			}
			req.matchedIngredients = []*model.IngredientAndVersion{ingredient}
			req.resolvedNamespace = ptr.To(model.ParseNamespace(*ingredient.Ingredient.PrimaryNamespace))
		}
	}

	// Now that we have the ingredient resolved we can also resolve the version requirement
	for _, req := range reqs {
		version := req.input.Version
		if req.input.Version == "" {
			continue
		}
		if _, err := strconv.Atoi(version); err == nil {
			// If the version number provided is a straight up integer (no dots or dashes) then assume it's a wildcard
			version = fmt.Sprintf("%d.x", version)
		}
		var err error
		req.resolvedVersionReq, err = bpModel.VersionStringToRequirements(version)
		if err != nil {
			return nil, errs.Wrap(err, "Could not process version string into requirements")
		}
	}

	return reqs, nil
}

func (i *Install) promptForMatchingIngredient(req *requirement) (*model.IngredientAndVersion, error) {
	if len(req.matchedIngredients) <= 1 {
		return nil, errs.New("promptForNamespace should never be called if there are no multiple ingredient matches")
	}

	choices := []string{}
	values := map[string]*model.IngredientAndVersion{}
	for _, i := range req.matchedIngredients {
		// Generate ingredient choices to present to the user
		name := fmt.Sprintf("%s (%s)", *i.Ingredient.Name, i.Ingredient.PrimaryNamespace)
		choices = append(choices, name)
		values[name] = i
	}

	// Prompt the user with the ingredient choices
	choice, err := i.prime.Prompt().Select(
		locale.Tl("prompt_pkgop_ingredient", "Multiple Matches"),
		locale.Tl("prompt_pkgop_ingredient_msg", "Your query has multiple matches. Which one would you like to use?"),
		choices, &choices[0],
	)
	if err != nil {
		return nil, errs.Wrap(err, "prompting failed")
	}

	// Return the user selected ingredient
	return values[choice], nil
}

func (i *Install) prepareBuildScript(script *buildscript.BuildScript, requirements requirements, ts time.Time) (*buildscript.BuildScript, error) {
	script.SetAtTime(ts)
	for _, req := range requirements {
		requirement := types.Requirement{
			Namespace:          req.resolvedNamespace.String(),
			Name:               req.input.Name,
			VersionRequirement: req.resolvedVersionReq,
		}

		err := script.AddRequirement(requirement)
		if err != nil {
			return nil, errs.Wrap(err, "Failed to update build expression with requirement")
		}
	}

	return script, nil
}

func (i *Install) updateCommitID(commitID strfmt.UUID) error {
	if err := localcommit.Set(i.prime.Project().Dir(), commitID.String()); err != nil {
		return locale.WrapError(err, "err_package_update_commit_id")
	}

	if i.prime.Config().GetBool(constants.OptinBuildscriptsConfig) {
		bp := bpModel.NewBuildPlannerModel(i.prime.Auth())
		script, err := bp.GetBuildScript(commitID.String())
		if err != nil {
			return errs.Wrap(err, "Could not get remote build expr and time")
		}

		err = buildscript_runbit.Update(i.prime.Project(), script)
		if err != nil {
			return locale.WrapError(err, "err_update_build_script")
		}
	}

	return nil
}

func IsBuildError(err error) bool {
	var errBuild *runtime.BuildError
	var errBuildPlanner *response.BuildPlannerError

	return errors.As(err, &errBuild) || errors.As(err, &errBuildPlanner)
}
