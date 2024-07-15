package dependencies

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/sliceutils"
	"github.com/ActiveState/cli/pkg/buildplan"
)

// showUpdatedPackages specifies whether or not to include updated dependencies in the direct
// dependencies list, and whether or not to include updated dependencies when calculating indirect
// dependency numbers.
const showUpdatedPackages = true

// OutputChangeSummary looks over the given build plans, and computes and lists the additional
// dependencies being installed for the requested packages, if any.
func OutputChangeSummary(out output.Outputer, newBuildPlan *buildplan.BuildPlan, oldBuildPlan *buildplan.BuildPlan) {
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

	// Process the existing runtime requirements into something we can easily compare against.
	alreadyInstalled := buildplan.Artifacts{}
	if oldBuildPlan != nil {
		alreadyInstalled = oldBuildPlan.Artifacts()
	}
	oldRequirements := alreadyInstalled.Ingredients().ToIDMap()

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
		oldVersion, exists := oldRequirements[ingredient.IngredientID]
		if exists && ingredient.Version != "" && oldVersion.Version != ingredient.Version {
			item = fmt.Sprintf("[ACTIONABLE]%s@%s[/RESET] → %s (%s)", oldVersion.Name, oldVersion.Version, item, locale.Tl("updated", "updated"))
		}

		out.Notice(fmt.Sprintf("  [DISABLED]%s[/RESET] %s", prefix, item))
	}

	out.Notice("") // blank line
}
