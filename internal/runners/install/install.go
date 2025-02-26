package install

import (
	"errors"
	"fmt"
	"regexp"
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

type resolvedRequirement struct {
	types.Requirement
	VersionLocale string `json:"version"` // VersionLocale represents the version as we want to show it to the user
	ingredient    *model.IngredientAndVersion
}

type requirement struct {
	Requested *captain.PackageValue `json:"requested"`
	Resolved  resolvedRequirement   `json:"resolved"`

	// Remainder are for display purposes only
	Type      model.NamespaceType `json:"type"`
	Operation types.Operation     `json:"operation"`
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
	bp := bpModel.NewBuildPlannerModel(i.prime.Auth(), i.prime.SvcModel())

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
	var oldCommit *bpModel.Commit
	var reqs requirements
	var ts time.Time
	{
		pg = output.StartSpinner(out, locale.T("progress_search"), constants.TerminalAnimationInterval)

		// Grab local commit info
		localCommitID, err := localcommit.Get(i.prime.Project().Dir())
		if err != nil {
			return errs.Wrap(err, "Unable to get local commit")
		}
		oldCommit, err = bp.FetchCommit(localCommitID, pj.Owner(), pj.Name(), nil)
		if err != nil {
			return errs.Wrap(err, "Failed to fetch old build result")
		}

		// Resolve timestamp, commit and languages used for current project.
		// This will be used to resolve the requirements.
		ts, err = commits_runbit.ExpandTimeForBuildScript(&params.Timestamp, i.prime.Auth(), oldCommit.BuildScript())
		if err != nil {
			return errs.Wrap(err, "Unable to get timestamp from params")
		}

		// Get languages used in current project
		languages, err := model.FetchLanguagesForBuildScript(oldCommit.BuildScript())
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

	if out.Type().IsStructured() {
		out.Print(output.Structured(reqs))
	} else {
		i.renderUserFacing(reqs)
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
	failed := []*requirement{}
	reqs := []*requirement{}
	for _, pkg := range packages {
		req := &requirement{Requested: pkg}
		if pkg.Namespace != "" {
			req.Resolved.Name = pkg.Name
			req.Resolved.Namespace = pkg.Namespace
		}

		// Find ingredients that match the pkg query
		ingredients, err := model.SearchIngredientsStrict(pkg.Namespace, pkg.Name, false, false, &ts, i.prime.Auth())
		if err != nil {
			return nil, locale.WrapError(err, "err_pkgop_search_err", "Failed to check for ingredients.")
		}

		// Filter out ingredients that don't target one of the supported languages
		if pkg.Namespace == "" {
			ingredients = sliceutils.Filter(ingredients, func(iv *model.IngredientAndVersion) bool {
				// Ensure namespace type matches
				if !model.NamespaceMatch(*iv.Ingredient.PrimaryNamespace, i.nsType.Matchable()) {
					return false
				}

				// Ensure that this is namespace covers one of the languages in our project
				// But only if we're aiming to install a package or bundle, because otherwise the namespace is not
				// guaranteed to hold the language.
				if i.nsType == model.NamespacePackage || i.nsType == model.NamespaceBundle {
					il := model.LanguageFromNamespace(*iv.Ingredient.PrimaryNamespace)
					for _, l := range languages {
						if l.Name == il {
							return true
						}
					}
					return false
				}
				return true
			})
		}

		// Resolve matched ingredients
		var ingredient *model.IngredientAndVersion
		if len(ingredients) == 1 {
			ingredient = ingredients[0]
		} else if len(ingredients) > 1 { // This wouldn't ever trigger if namespace was provided as that should guarantee a single result
			var err error
			ingredient, err = i.promptForMatchingIngredient(req, ingredients)
			if err != nil {
				return nil, errs.Wrap(err, "Prompting for namespace failed")
			}
		}
		if ingredient == nil {
			failed = append(failed, req)
		} else {
			req.Resolved.Name = ingredient.Ingredient.NormalizedName
			req.Resolved.Namespace = *ingredient.Ingredient.PrimaryNamespace
			req.Resolved.ingredient = ingredient
		}

		reqs = append(reqs, req)
	}

	// Fail if not all requirements could be resolved
	if len(failed) > 0 {
		return nil, errNoMatches{error: errs.New("Failed to resolve requirements"), requirements: failed, languages: languages}
	}

	// Now that we have the ingredient resolved we can also resolve the version requirement.
	// We can also set the type and operation, which are used for conveying what happened to the user.
	for _, req := range reqs {
		// Set requirement type
		req.Type = model.ParseNamespace(req.Resolved.Namespace).Type()

		if err := resolveVersion(req); err != nil {
			return nil, errs.Wrap(err, "Could not resolve version")
		}
	}

	return reqs, nil
}

var versionRe = regexp.MustCompile(`^\d(\.\d+)*$`)

func resolveVersion(req *requirement) error {
	version := req.Requested.Version

	// An empty version means "Auto"
	if req.Requested.Version == "" {
		req.Resolved.VersionLocale = locale.T("constraint_auto")
		return nil
	}

	// Verify that the version provided can be resolved
	if versionRe.MatchString(version) {
		match := false
		for _, knownVersion := range req.Resolved.ingredient.Versions {
			if knownVersion.Version == version {
				match = true
				break
			}
		}
		if !match {
			for _, knownVersion := range req.Resolved.ingredient.Versions {
				if strings.HasPrefix(knownVersion.Version, version) {
					version = version + ".x" // The user supplied a partial version, resolve it as a wildcard
				}
			}
		}
	}

	var err error
	req.Resolved.VersionLocale = version
	req.Resolved.VersionRequirement, err = bpModel.VersionStringToRequirements(version)
	if err != nil {
		return errs.Wrap(err, "Could not process version string into requirements")
	}

	return nil
}

func (i *Install) promptForMatchingIngredient(req *requirement, ingredients []*model.IngredientAndVersion) (*model.IngredientAndVersion, error) {
	if len(ingredients) <= 1 {
		return nil, errs.New("promptForNamespace should never be called if there are no multiple ingredient matches")
	}

	choices := []string{}
	values := map[string]*model.IngredientAndVersion{}
	for _, i := range ingredients {
		// Generate ingredient choices to present to the user
		name := fmt.Sprintf("%s (%s)", *i.Ingredient.Name, *i.Ingredient.PrimaryNamespace)
		choices = append(choices, name)
		values[name] = i
	}

	// Prompt the user with the ingredient choices
	choice, err := i.prime.Prompt().Select(
		locale.T("prompt_pkgop_ingredient"),
		locale.Tr("prompt_pkgop_ingredient_msg", req.Requested.String()),
		choices, &choices[0], nil,
	)
	if err != nil {
		return nil, errs.Wrap(err, "prompting failed")
	}

	// Return the user selected ingredient
	return values[choice], nil
}

func (i *Install) renderUserFacing(reqs requirements) {
	for _, req := range reqs {
		l := "install_report_added"
		if req.Operation == types.OperationUpdated {
			l = "install_report_updated"
		}
		i.prime.Output().Notice(locale.Tr(l, fmt.Sprintf("%s/%s@%s", req.Resolved.Namespace, req.Resolved.Name, req.Resolved.VersionLocale)))
	}
	i.prime.Output().Notice("")
}

func prepareBuildScript(script *buildscript.BuildScript, requirements requirements, ts time.Time) error {
	script.SetAtTime(ts, true)
	for _, req := range requirements {
		requirement := types.Requirement{
			Namespace:          req.Resolved.Namespace,
			Name:               req.Requested.Name,
			VersionRequirement: req.Resolved.VersionRequirement,
		}

		req.Operation = types.OperationUpdated
		if err := script.RemoveRequirement(requirement); err != nil {
			if !errors.As(err, ptr.To(&buildscript.RequirementNotFoundError{})) {
				return errs.Wrap(err, "Could not remove requirement")
			}
			req.Operation = types.OperationAdded // If req could not be found it means this is an addition
		}

		err := script.AddRequirement(requirement)
		if err != nil {
			return errs.Wrap(err, "Failed to update build expression with requirement")
		}
	}

	return nil
}
