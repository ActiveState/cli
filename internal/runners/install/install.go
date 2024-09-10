package install

import (
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
	"github.com/ActiveState/cli/internal/runbits/commits_runbit"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/internal/runbits/reqop_runbit"
	"github.com/ActiveState/cli/internal/runbits/runtime/trigger"
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

// Params tracks the info required for running Install.
type Params struct {
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
	prime  primeable
	nsType model.NamespaceType
}

// New prepares an installation execution context for use.
func New(prime primeable, nsType model.NamespaceType) *Install {
	return &Install{prime, nsType}
}

// Run executes the install behavior.
func (i *Install) Run(params Params) (rerr error) {
	defer i.rationalizeError(&rerr)

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
		pg = nil
	}

	// Prepare updated buildscript
	script := oldCommit.BuildScript()
	if err := prepareBuildScript(script, reqs, ts); err != nil {
		return errs.Wrap(err, "Could not prepare build script")
	}

	// Update local checkout and source runtime changes
	if err := reqop_runbit.UpdateAndReload(i.prime, script, oldCommit, locale.Tr("commit_message_added", reqs.String()), trigger.TriggerInstall); err != nil {
		return errs.Wrap(err, "Failed to update local checkout")
	}

	// All done
	out.Notice(locale.T("operation_success_local"))

	return nil
}

type errNoMatches struct {
	error
	requirements []*requirement
	languages    []model.Language
}

// resolveRequirements will attempt to resolve the ingredient and namespace for each requested package
func (i *Install) resolveRequirements(packages captain.PackagesValue, ts time.Time, languages []model.Language) (requirements, error) {
	var disambiguate []*requirement
	var failed []*requirement
	reqs := []*requirement{}
	for _, pkg := range packages {
		req := &requirement{input: pkg}
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
			ingredients = sliceutils.Filter(ingredients, func(iv *model.IngredientAndVersion) bool {
				if !model.NamespaceMatch(*iv.Ingredient.PrimaryNamespace, i.nsType.Matchable()) {
					return false
				}
				il := model.LanguageFromNamespace(*iv.Ingredient.PrimaryNamespace)
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
		return nil, errNoMatches{error: errs.New("Failed to resolve requirements"), requirements: failed, languages: languages}
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

func prepareBuildScript(script *buildscript.BuildScript, requirements requirements, ts time.Time) error {
	script.SetAtTime(ts)
	for _, req := range requirements {
		requirement := types.Requirement{
			Namespace:          req.resolvedNamespace.String(),
			Name:               req.input.Name,
			VersionRequirement: req.resolvedVersionReq,
		}

		err := script.AddRequirement(requirement)
		if err != nil {
			return errs.Wrap(err, "Failed to update build expression with requirement")
		}
	}

	return nil
}
