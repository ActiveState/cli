package dependencies

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/sliceutils"
	"github.com/ActiveState/cli/pkg/buildplan"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
	"github.com/ActiveState/cli/pkg/platform/model/buildplanner"
)

type primeable interface {
	primer.Outputer
	primer.Auther
	primer.Projecter
}

// showUpdatedPackages specifies whether or not to include updated dependencies in the direct
// dependencies list, and whether or not to include updated dependencies when calculating indirect
// dependency numbers.
const showUpdatedPackages = true

func OutputChangeSummary(prime primeable, rtCommit *buildplanner.Commit, oldBuildPlan *buildplan.BuildPlan) {
	if expr, err := json.Marshal(rtCommit.BuildScript()); err == nil {
		bpm := buildplanner.NewBuildPlannerModel(prime.Auth())
		params := &buildplanner.ImpactReportParams{
			Owner:          prime.Project().Owner(),
			Project:        prime.Project().Name(),
			BeforeCommitId: rtCommit.ParentID,
			AfterExpr:      expr,
		}
		if impactReport, err := bpm.ImpactReport(params); err == nil {
			outputChangeSummaryFromImpactReport(prime.Output(), rtCommit.BuildPlan(), impactReport)
			return
		} else {
			multilog.Error("Failed to fetch impact report: %v", err)
		}
	} else {
		multilog.Error("Failed to marshal buildexpression: %v", err)
	}
	outputChangeSummaryFromBuildPlans(prime.Output(), rtCommit.BuildPlan(), oldBuildPlan)
}

// outputChangeSummaryFromBuildPlans looks over the given build plans, and computes and lists the
// additional dependencies being installed for the requested packages, if any.
func outputChangeSummaryFromBuildPlans(out output.Outputer, newBuildPlan *buildplan.BuildPlan, oldBuildPlan *buildplan.BuildPlan) {
	requested := newBuildPlan.RequestedArtifacts().ToIDMap()

	addedString := []string{}
	addedLocale := []string{}
	dependencies := buildplan.Ingredients{}
	directDependencies := buildplan.Ingredients{}
	changeset := newBuildPlan.DiffArtifacts(oldBuildPlan, false)
	for _, a := range changeset.Added {
		if _, exists := requested[a.ArtifactID]; exists {
			v := fmt.Sprintf("%s@%s", a.Name(), a.Version())
			addedString = append(addedLocale, v)
			addedLocale = append(addedLocale, fmt.Sprintf("[ACTIONABLE]%s[/RESET]", v))

			for _, i := range a.Ingredients {
				dependencies = append(dependencies, i.RuntimeDependencies(true)...)
				directDependencies = append(directDependencies, i.RuntimeDependencies(false)...)
			}
		}
	}

	alreadyInstalledVersions := map[strfmt.UUID]string{}
	if oldBuildPlan != nil {
		for _, a := range oldBuildPlan.Artifacts() {
			alreadyInstalledVersions[a.ArtifactID] = a.Version()
		}
	}

	outputChangeSummary(out, addedString, addedLocale, dependencies, directDependencies, alreadyInstalledVersions)
}

func outputChangeSummaryFromImpactReport(out output.Outputer, buildPlan *buildplan.BuildPlan, report *response.ImpactReportResult) {
	alreadyInstalledVersions := map[strfmt.UUID]string{}
	addedString := []string{}
	addedLocale := []string{}
	dependencies := buildplan.Ingredients{}
	directDependencies := buildplan.Ingredients{}
	for _, i := range report.Ingredients {
		if i.Before != nil {
			alreadyInstalledVersions[strfmt.UUID(i.Before.IngredientID)] = i.Before.Version
		}

		if i.After == nil || !i.After.IsRequirement {
			continue
		}

		if i.Before == nil {
			v := fmt.Sprintf("%s@%s", i.Name, i.After.Version)
			addedString = append(addedLocale, v)
			addedLocale = append(addedLocale, fmt.Sprintf("[ACTIONABLE]%s[/RESET]", v))
		}

		for _, bpi := range buildPlan.Ingredients() {
			if bpi.IngredientID != strfmt.UUID(i.After.IngredientID) {
				continue
			}
			dependencies = append(dependencies, bpi.RuntimeDependencies(true)...)
			directDependencies = append(directDependencies, bpi.RuntimeDependencies(false)...)
		}
	}

	outputChangeSummary(out, addedString, addedLocale, dependencies, directDependencies, alreadyInstalledVersions)
}

func outputChangeSummary(
	out output.Outputer,
	addedString []string,
	addedLocale []string,
	dependencies buildplan.Ingredients,
	directDependencies buildplan.Ingredients,
	alreadyInstalledVersions map[strfmt.UUID]string,
) {
	dependencies = sliceutils.UniqueByProperty(dependencies, func(i *buildplan.Ingredient) any { return i.IngredientID })
	directDependencies = sliceutils.UniqueByProperty(directDependencies, func(i *buildplan.Ingredient) any { return i.IngredientID })
	commonDependencies := directDependencies.CommonRuntimeDependencies().ToIDMap()
	numIndirect := len(dependencies) - len(directDependencies) - len(commonDependencies)

	sort.SliceStable(directDependencies, func(i, j int) bool {
		return directDependencies[i].Name < directDependencies[j].Name
	})

	logging.Debug("packages %s have %d direct dependencies and %d indirect dependencies",
		strings.Join(addedString, ", "), len(directDependencies), numIndirect)
	if len(directDependencies) == 0 {
		return
	}

	localeKey := "additional_dependencies"
	if numIndirect > 0 {
		localeKey = "additional_total_dependencies"
	}
	out.Notice("   " + locale.Tr(localeKey, strings.Join(addedLocale, ", "), strconv.Itoa(len(directDependencies)), strconv.Itoa(numIndirect)))

	// A direct dependency list item is of the form:
	//   ├─ name@version (X dependencies)
	// or
	//   └─ name@oldVersion → name@newVersion (Updated)
	// depending on whether or not it has subdependencies, and whether or not showUpdatedPackages is
	// `true`.
	for i, ingredient := range directDependencies {
		prefix := " ├─"
		if i == len(directDependencies)-1 {
			prefix = " └─"
		}

		// Retrieve runtime dependencies, and then filter out any dependencies that are common between all added ingredients.
		runtimeDeps := ingredient.RuntimeDependencies(true)
		runtimeDeps = runtimeDeps.Filter(func(i *buildplan.Ingredient) bool { _, ok := commonDependencies[i.IngredientID]; return !ok })

		subdependencies := ""
		if numSubs := len(runtimeDeps); numSubs > 0 {
			subdependencies = fmt.Sprintf(" ([ACTIONABLE]%s[/RESET] dependencies)", // intentional leading space
				strconv.Itoa(numSubs))
		}

		item := fmt.Sprintf("[ACTIONABLE]%s@%s[/RESET]%s", // intentional omission of space before last %s
			ingredient.Name, ingredient.Version, subdependencies)
		oldVersion, exists := alreadyInstalledVersions[ingredient.IngredientID]
		if exists && ingredient.Version != "" && oldVersion != ingredient.Version {
			item = fmt.Sprintf("[ACTIONABLE]%s@%s[/RESET] → %s (%s)", ingredient.Name, oldVersion, item, locale.Tl("updated", "updated"))
		}

		out.Notice(fmt.Sprintf("  [DISABLED]%s[/RESET] %s", prefix, item))
	}

	out.Notice("") // blank line
}
