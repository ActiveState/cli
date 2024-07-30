package dependencies

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/sliceutils"
	"github.com/ActiveState/cli/pkg/buildplan"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
	"github.com/ActiveState/cli/pkg/platform/model/buildplanner"
)

// showUpdatedPackages specifies whether or not to include updated dependencies in the direct
// dependencies list, and whether or not to include updated dependencies when calculating indirect
// dependency numbers.
const showUpdatedPackages = true

func OutputChangeSummary(out output.Outputer, report *response.ImpactReportResult, rtCommit *buildplanner.Commit) {
	// Process the impact report, looking for package additions or updates.
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

		for _, bpi := range rtCommit.BuildPlan().Ingredients() {
			if bpi.IngredientID != strfmt.UUID(i.After.IngredientID) {
				continue
			}
			dependencies = append(dependencies, bpi.RuntimeDependencies(true)...)
			directDependencies = append(directDependencies, bpi.RuntimeDependencies(false)...)
		}
	}

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

	// Output a summary of changes.
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
