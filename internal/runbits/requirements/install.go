package requirements

import (
	"fmt"
	"strings"
	"time"

	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/internal/runbits/cves"
	"github.com/ActiveState/cli/internal/runbits/dependencies"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/internal/runbits/runtime"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	"github.com/ActiveState/cli/pkg/platform/model"
	bpModel "github.com/ActiveState/cli/pkg/platform/model/buildplanner"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
)

type ErrMultipleMatches struct {
	Query   string
	Matches []*model.IngredientAndVersion
}

func (e ErrMultipleMatches) Error() string {
	return "multiple matches"
}

func (r *RequirementOperation) Install(ts *time.Time, requirements ...*Requirement) (rerr error) {
	defer r.rationalizeError(&rerr)

	if len(requirements) == 0 {
		return errNoRequirements
	}
	if r.Project == nil {
		return rationalize.ErrNoProject
	}
	if r.Project.IsHeadless() {
		return rationalize.ErrHeadless
	}

	r.Output.Notice(locale.Tr("operating_message", r.Project.NamespaceString(), r.Project.Dir()))

	commitID, err := localcommit.Get(r.Project.Dir())
	if err != nil {
		return errs.Wrap(err, "Unable to get local commit")
	}

	// Construct a list of namespaces to search for ingredients in (if unspecified).
	// This consists of the language namespace as well as namespaces for each of a project's
	// language(s).
	namespaces := []*model.Namespace{
		ptr.To(model.NewNamespaceLanguage()),
	}
	if languages, err := model.FetchLanguagesForCommit(commitID, r.Auth); err == nil {
		for _, lang := range languages {
			namespaces = append(namespaces, ptr.To(model.NewNamespacePackage(lang.Name)))
			namespaces = append(namespaces, ptr.To(model.NewNamespaceBundle(lang.Name)))
		}
	} else {
		logging.Debug("Could not get languages from project: %v", err)
	}

	// Search for the requested requirements.
	ingredients, err := r.searchForRequirements(requirements, namespaces)
	if err != nil {
		return errs.Wrap(err, "Failed to search for requirements")
	}

	// Start the process of creating a commit with the requested changes.
	bp := bpModel.NewBuildPlannerModel(r.Auth)

	pg := output.StartSpinner(r.Output, locale.T("progress_commit"), constants.TerminalAnimationInterval)
	defer func() {
		if pg != nil {
			pg.Stop(locale.T("progress_fail"))
		}
	}()

	// Prepare a build script with the requested changes.
	script, err := bp.GetBuildScript(commitID.String())
	if err != nil {
		return errs.Wrap(err, "Failed to get build script")
	}

	if ts == nil {
		// If no atTime was provided then we need to ensure that the atTime in the script is updated to
		// use the most recent, which is either the current value or the platform latest.
		latest, err := model.FetchLatestTimeStamp(r.Auth)
		if err != nil {
			return errs.Wrap(err, "Unable to fetch latest Platform timestamp")
		}
		atTime := script.AtTime()
		if atTime == nil || latest.After(*atTime) {
			ts = &latest
		}
	}
	if ts != nil {
		script.SetAtTime(*ts)
	}

	// Add or update requirements in the build script.
	for i, ing := range ingredients {
		// Determine if the ingredient is being added or updated.
		if existingReq, err := model.GetRequirement(
			commitID,
			model.NamespaceFromIngredient(ing),
			*ing.Ingredient.Name,
			r.Auth,
		); err == nil && existingReq != nil {
			requirements[i].Operation = types.OperationUpdated // modification of input is intentional
		} else if err != nil {
			return errs.Wrap(err, "Failed to check for existing requirement")
		}

		// Determine the ingredient's version requirements.
		version := requirements[i].Version
		if strings.ToLower(version) == latestVersion {
			version = ""
		} else if versionRe.MatchString(version) {
			// If a bare version number was given, and if it is a partial version number
			// (e.g. requests@2), we'll want to ultimately append a '.x' suffix.
			for _, knownVersion := range ing.Versions {
				if knownVersion.Version == version {
					break
				} else if strings.HasPrefix(knownVersion.Version, version) {
					version += ".x"
				}
			}
		}
		versionReqs, err := bpModel.VersionStringToRequirements(version)
		if err != nil {
			return errs.Wrap(err, "Could not process version string into requirements")
		}

		// Update the build script with the ingredient.
		err = script.UpdateRequirement(requirements[i].Operation, types.Requirement{
			Name:               *ing.Ingredient.Name,
			Namespace:          *ing.Ingredient.PrimaryNamespace,
			VersionRequirement: versionReqs,
			Revision:           requirements[i].Revision,
		})
		if err != nil {
			return errs.Wrap(err, "Failed to add ingredient to requirements")
		}
	}

	// Stage the commit.
	commitMessages := make([]string, len(ingredients))
	for i, ing := range ingredients {
		req := requirements[i]
		message := packageCommitMessage(req.Operation, req.Name, req.Version)
		if model.NamespaceFromIngredient(ing).Type() == model.NamespaceLanguage {
			message = languageCommitMessage(req.Operation, req.Name, req.Version)
		}
		commitMessages[i] = message
	}
	commitMessage := commitMessages[0]
	if len(commitMessages) > 1 {
		locale.Tl("commit_message_multiple",
			"Committing changes to multiple requirements: {{.V0}}",
			strings.Join(commitMessages, ", "))
	}
	params := bpModel.StageCommitParams{
		Owner:        r.Project.Owner(),
		Project:      r.Project.Name(),
		ParentCommit: commitID.String(),
		Description:  commitMessage,
		Script:       script,
	}
	stagedCommitID, err := bp.StageCommit(params)
	if err != nil {
		return locale.WrapError(err, "err_package_save_and_build", "Error occurred while trying to create a commit")
	}

	pg.Stop(locale.T("progress_success"))
	pg = nil

	// Solve the runtime.
	rt, rtCommit, err := runtime.Solve(r.Auth, r.Output, r.Analytics, r.Project, &stagedCommitID, target.TriggerPackage, r.SvcModel, r.Config, runtime.OptNone)
	if err != nil {
		return errs.Wrap(err, "Could not solve runtime")
	}

	// Output change summary.
	previousCommit, err := bp.FetchCommit(commitID, r.Project.Owner(), r.Project.Name(), nil)
	if err != nil {
		return errs.Wrap(err, "Failed to fetch build result for previous commit")
	}
	oldBuildPlan := previousCommit.BuildPlan()
	r.Output.Notice("") // blank line
	dependencies.OutputChangeSummary(r.Output, rtCommit.BuildPlan(), oldBuildPlan)

	// Report CVEs.
	if err := cves.NewCveReport(r.prime).Report(rtCommit.BuildPlan(), oldBuildPlan); err != nil {
		return errs.Wrap(err, "Could not report CVEs")
	}

	// Update the runtime.
	if !r.Config.GetBool(constants.AsyncRuntimeConfig) {
		r.Output.Notice("")
		if !rt.HasCache() {
			r.Output.Notice(output.Title(locale.T("install_runtime")))
			r.Output.Notice(locale.T("install_runtime_info"))
		} else {
			r.Output.Notice(output.Title(locale.T("update_runtime")))
			r.Output.Notice(locale.T("update_runtime_info"))
		}

		// refresh or install runtime
		err = runtime.UpdateByReference(rt, rtCommit, r.Auth, r.Project, r.Output, r.Config, runtime.OptMinimalUI)
		if err != nil {
			if !runbits.IsBuildError(err) {
				// If the error is not a build error we want to retain the changes
				if err2 := r.updateCommitID(stagedCommitID); err2 != nil {
					return errs.Pack(err, locale.WrapError(err2, "err_package_update_commit_id"))
				}
			}
			return errs.Wrap(err, "Failed to refresh runtime")
		}
	}

	// Record the new commit ID and update the local build script.
	if err := r.updateCommitID(stagedCommitID); err != nil {
		return locale.WrapError(err, "err_package_update_commit_id")
	}

	// Output overall summary.
	messages := make([]string, len(ingredients))
	jsonObjects := make([]interface{}, len(ingredients))
	for i, ing := range ingredients {
		name := *ing.Ingredient.Name
		nsType := "package"
		switch model.NamespaceFromIngredient(ing).Type() {
		case model.NamespaceLanguage:
			nsType = "language"
		}
		req := requirements[i]
		message := locale.Tr(fmt.Sprintf("%s_version_%s", nsType, req.Operation), name, req.Version)
		if req.Version == "" {
			message = locale.Tr(fmt.Sprintf("%s_%s", nsType, req.Operation), name)
		}
		messages = append(messages, message)
		if req.Name != name {
			messages = append(messages, locale.Tl("package_version_differs",
				"Note: the actual package name ({{.V0}}) is different from the requested package name ({{.V1}})",
				name, req.Name))
		}
		jsonObjects[i] = &struct {
			Name      string `json:"name"`
			Version   string `json:"version,omitempty"`
			Type      string `json:"type"`
			Operation string `json:"operation"`
		}{
			name,
			req.Version,
			nsType,
			req.Operation.String(),
		}
	}
	r.Output.Print(output.Prepare(strings.Join(messages, "\n"), jsonObjects))

	r.Output.Notice("") // blank line
	r.Output.Notice(locale.T("operation_success_local"))

	return nil
}

// searchForRequirements searches for the given requirements in the given namespaces and returns
// the associated list of found ingredients.
func (r *RequirementOperation) searchForRequirements(requirements []*Requirement, namespaces []*model.Namespace) (results []*model.IngredientAndVersion, rerr error) {
	results = make([]*model.IngredientAndVersion, len(requirements))

	names := make([]string, len(requirements))
	for i, req := range requirements {
		names[i] = req.Name
	}

	pg := output.StartSpinner(r.Output, locale.Tr("progress_search", strings.Join(names, ", ")), constants.TerminalAnimationInterval)
	defer func() {
		if rerr != nil {
			pg.Stop(locale.T("progress_fail"))
		}
	}()

	for i, req := range requirements {
		ingredients, err := r.searchForIngredient(req, req.Namespace)
		if err != nil {
			return nil, errs.Wrap(err, "Unable to search for requirement '%s'", req.Name)
		}

		// Collect all ingredients into a unique list of requirement names and namespaces.
		unique := make([]*model.IngredientAndVersion, 0)
		for _, i := range ingredients {
			// If the ingredient's requirement did not specify a namespace, verify that ingredient is in
			// one of the requested namespaces. The ingredient API does not support searching in a set of
			// specific namespaces. It's either one or all.
			isValidNamespace := req.Namespace != nil
			if !isValidNamespace {
				for _, ns := range namespaces {
					if ns.String() == *i.Ingredient.PrimaryNamespace {
						isValidNamespace = true
						break
					}
				}
			}
			if isValidNamespace && !funk.Contains(unique, i) {
				unique = append(unique, i)
			}
		}

		if len(unique) == 0 {
			suggestions, err := getSuggestions(req.Namespace, req.Name, namespaces, r.Auth)
			if err != nil {
				multilog.Error("Failed to retrieve suggestions: %v", err)
			}

			if len(suggestions) == 0 {
				return nil, &ErrNoMatches{
					locale.WrapExternalError(err, "package_ingredient_alternatives_nosuggest", "", req.Name),
					req.Name, nil}
			}

			return nil, &ErrNoMatches{
				locale.WrapExternalError(err, "package_ingredient_alternatives", "", req.Name, strings.Join(suggestions, "\n")),
				req.Name, ptr.To(strings.Join(suggestions, "\n"))}
		}

		if len(unique) > 1 {
			return nil, &ErrMultipleMatches{req.Name, unique}
		}

		results[i] = unique[0]
	}

	pg.Stop(locale.T("progress_found"))
	return results, nil
}

func (r *RequirementOperation) searchForIngredient(req *Requirement, namespace *model.Namespace) ([]*model.IngredientAndVersion, error) {
	name := req.Name
	ns := ""
	caseSensitive := false

	if namespace != nil {
		if normalized, err := model.FetchNormalizedName(*namespace, name, r.Auth); err == nil {
			name = normalized
		} else {
			multilog.Error("Failed to normalize '%s': %v", req.Name, err)
		}
		ns = namespace.String()
		//caseSensitive = true // ideally case-sensitive would be true (PB-4371)
	}

	return model.SearchIngredientsStrict(ns, name, caseSensitive, false, nil, r.Auth)
}
