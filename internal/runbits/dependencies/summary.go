package dependencies

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildplan"
)

// OutputSummary lists the given runtime dependencies being installed along with their
// subdependencies (if any).
func OutputSummary(out output.Outputer, directDependencies []artifact.ArtifactID, artifacts artifact.Map) {
	if len(directDependencies) == 0 {
		return
	}

	dependencies := make(map[artifact.ArtifactID][]artifact.ArtifactID)
	for _, artifactId := range directDependencies {
		subdependencies := buildplan.RecursiveDependenciesFor(artifactId, artifacts)
		dependencies[artifactId] = subdependencies
	}
	sort.SliceStable(directDependencies, func(i, j int) bool {
		return artifacts[directDependencies[i]].Name < artifacts[directDependencies[j]].Name
	})

	out.Notice("") // blank line

	out.Notice(locale.Tl("setting_up_dependencies", "Setting up the following dependencies:"))

	for i, artifactId := range directDependencies {
		prefix := "├─"
		if i == len(directDependencies)-1 {
			prefix = "└─"
		}
		dep := artifacts[artifactId]

		version := ""
		if dep.Version != nil {
			version = *dep.Version
		}

		subdependencies := ""
		if numSubs := len(dependencies[dep.ArtifactID]); numSubs > 0 {
			subdependencies = fmt.Sprintf("([ACTIONABLE]%s[/RESET] dependencies)", strconv.Itoa(numSubs))
		}

		item := fmt.Sprintf("[ACTIONABLE]%s@%s[/RESET] %s", dep.Name, version, subdependencies)

		out.Notice(fmt.Sprintf("[DISABLED]%s[/RESET] %s", prefix, item))
	}

	out.Notice("") // blank line
}
