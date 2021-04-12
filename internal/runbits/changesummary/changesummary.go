package changesummary

import (
	"fmt"
	"strconv"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/thoas/go-funk"
)

// ChangeSummary prints the summary of changes to the encapsulated outputer
type ChangeSummary struct {
	out output.Outputer
}

func New(out output.Outputer) *ChangeSummary {
	return &ChangeSummary{out}
}

// ChangeSummary currently only write a summary if exactly one package has been added (after `state install`)
// In this case, it will print
// - the number of direct dependencies this package brings in,
// - the total number of new dependencies
// - the names of the direct dependencies (and the count of their sub-dependencies)
func (cs *ChangeSummary) ChangeSummary(artifacts artifact.ArtifactRecipeMap, requested artifact.ArtifactChangeset, changed artifact.ArtifactChangeset) error {
	// currently we only print a change summary if we are adding exactly ONE package
	if len(requested.Added) != 1 {
		return nil
	}

	ar, ok := artifacts[requested.Added[0]]
	if !ok {
		return errs.New("Did not find requested artifact in ArtifactRecipeMap")
	}

	// the added (direct dependencies) of this artifact that are actually new in this project
	addedDependencies := funk.Join(ar.Dependencies, changed.Added, funk.InnerJoin).([]artifact.ArtifactID)

	cs.out.Notice("")
	cs.out.Notice(locale.Tl(
		"changesummary_title",
		"[NOTICE]{{.V0}}[/RESET] includes {{.V1}} dependencies, for a combined total of {{.V2}} new dependencies.",
		ar.Name, strconv.Itoa(len(addedDependencies)), strconv.Itoa(len(changed.Added)),
	))
	for i, dep := range addedDependencies {
		depMapping, ok := artifacts[dep]
		if !ok {
			logging.Error("Could not find dependency %s in artifactsMap", dep)
			continue
		}
		var depCount string
		recDeps := artifact.RecursiveDependenciesFor(dep, artifacts)
		filteredRecDeps := funk.Join(recDeps, changed.Added, funk.InnerJoin).([]artifact.ArtifactID)
		if len(filteredRecDeps) > 0 {
			depCount = locale.Tl("ingredient_dependency_count", " ({{.V0}} dependencies)", strconv.Itoa(len(filteredRecDeps)))
		}
		prefix := "├─"
		if i == len(addedDependencies)-1 {
			prefix = "└─"
		}
		cs.out.Notice(fmt.Sprintf("  [DISABLED]%s[/RESET] %s%s", prefix, depMapping.Name, depCount))
	}
	return nil
}
