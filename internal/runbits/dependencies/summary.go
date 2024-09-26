package dependencies

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/buildplan"
)

// OutputSummary lists the given runtime dependencies being installed along with their
// subdependencies (if any).
func OutputSummary(out output.Outputer, directDependencies buildplan.Artifacts) {
	if len(directDependencies) == 0 {
		return
	}

	ingredients := directDependencies.Filter(buildplan.FilterStateArtifacts()).Ingredients()
	commonDependencies := ingredients.CommonRuntimeDependencies().ToIDMap()

	sort.SliceStable(ingredients, func(i, j int) bool {
		return ingredients[i].Name < ingredients[j].Name
	})

	out.Notice("") // blank line
	out.Notice(locale.Tl("setting_up_dependencies", "  Setting up the following dependencies:"))

	for i, ingredient := range ingredients {
		prefix := "  ├─"
		if i == len(ingredients)-1 {
			prefix = "  └─"
		}

		subDependencies := ingredient.RuntimeDependencies(true)
		if _, isCommon := commonDependencies[ingredient.IngredientID]; !isCommon {
			// If the ingredient is itself not a common sub-dependency; filter out any common sub dependencies so we don't
			// report counts multiple times.
			subDependencies = subDependencies.Filter(buildplan.FilterOutIngredients{commonDependencies}.Filter)
		}
		subdepLocale := ""
		if numSubs := len(subDependencies); numSubs > 0 {
			subdepLocale = locale.Tl("summary_subdeps", "([ACTIONABLE]{{.V0}}[/RESET] sub-dependencies)", strconv.Itoa(numSubs))
		}

		item := fmt.Sprintf("[ACTIONABLE]%s@%s[/RESET] %s", ingredient.Name, ingredient.Version, subdepLocale)

		out.Notice(fmt.Sprintf("[DISABLED]%s[/RESET] %s", prefix, item))
	}

	out.Notice("") // blank line
}
