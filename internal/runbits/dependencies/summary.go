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

	ingredients := directDependencies.Ingredients()

	sort.SliceStable(ingredients, func(i, j int) bool {
		return ingredients[i].Name < ingredients[j].Name
	})

	out.Notice("") // blank line

	out.Notice(locale.Tl("setting_up_dependencies", "Setting up the following dependencies:"))

	for i, ingredient := range ingredients {
		prefix := "├─"
		if i == len(directDependencies)-1 {
			prefix = "└─"
		}

		subdependencies := ""
		if numSubs := len(ingredient.Dependencies(true)); numSubs > 0 {
			subdependencies = fmt.Sprintf("([ACTIONABLE]%s[/RESET] dependencies)", strconv.Itoa(numSubs))
		}

		item := fmt.Sprintf("[ACTIONABLE]%s@%s[/RESET] %s", ingredient.Name, ingredient.Version, subdependencies)

		out.Notice(fmt.Sprintf("[DISABLED]%s[/RESET] %s", prefix, item))
	}

	out.Notice("") // blank line
}
