package dependencies

import (
	"fmt"
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

// OutputChangeSummary looks over the given artifact changeset and attempts to determine if a single
// package install request was made. If so, it computes and lists the additional dependencies being
// installed for that package.
// `artifacts` is an ArtifactMap containing artifacts in the changeset, and `filter` contains any
// runtime requirements/artifacts already installed.
func OutputChangeSummary(out output.Outputer, changeset *buildplan.ArtifactChangeset, alreadyInstalled buildplan.Artifacts) {
	addedString := []string{}
	addedLocale := []string{}
	added := buildplan.Ingredients{}
	dependencies := buildplan.Ingredients{}
	directDependencies := buildplan.Ingredients{}
	for _, a := range changeset.Added {
		added = append(added, a.Ingredients...)
		for _, i := range a.Ingredients {
			addedString = append(addedLocale, fmt.Sprintf("%s@%s", i.Name, i.Version))
			addedLocale = append(addedLocale, fmt.Sprintf("[ACTIONABLE]%s[/RESET]", addedString))
			dependencies = append(dependencies, i.RuntimeDependencies(true)...)
			directDependencies = append(dependencies, i.RuntimeDependencies(false)...)
		}
	}

	dependencies = sliceutils.UniqueByProperty(dependencies, func(i *buildplan.Ingredient) any { return i.IngredientID })
	directDependencies = sliceutils.UniqueByProperty(directDependencies, func(i *buildplan.Ingredient) any { return i.IngredientID })
	numIndirect := len(dependencies) - len(directDependencies)

	logging.Debug("packages %s have %d direct dependencies and %d total, unique dependencies",
		strings.Join(addedString, ", "), len(directDependencies), numIndirect)
	if len(directDependencies) == 0 {
		return
	}

	// Process the existing runtime requirements into something we can easily compare against.
	oldRequirements := alreadyInstalled.Ingredients().ToIDMap()

	localeKey := "additional_dependencies"
	if numIndirect > 0 {
		localeKey = "additional_total_dependencies"
	}
	out.Notice(locale.Tr(localeKey,
		strings.Join(addedLocale, ", "), strconv.Itoa(len(dependencies)), strconv.Itoa(numIndirect)))

	// A direct dependency list item is of the form:
	//   ├─ name@version (X dependencies)
	// or
	//   └─ name@oldVersion → name@newVersion (Updated)
	// depending on whether or not it has subdependencies, and whether or not showUpdatedPackages is
	// `true`.
	for i, ingredient := range directDependencies {
		prefix := "├─"
		if i == len(directDependencies)-1 {
			prefix = "└─"
		}

		ingredientDeps := ingredient.RuntimeDependencies(true)
		subdependencies := ""
		if numSubs := len(ingredientDeps); numSubs > 0 {
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
