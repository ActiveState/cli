package dependencies

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildplan"
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
func OutputChangeSummary(out output.Outputer, changeset artifact.ArtifactChangeset, artifacts artifact.Map, filter artifact.Map) {
	// Determine which package was installed.
	var addedId *artifact.ArtifactID
	for _, candidateId := range changeset.Added {
		if !isDependency(candidateId, changeset, artifacts) {
			if addedId != nil {
				return // more than two independent packages were added
			}
			foundId := candidateId
			addedId = &foundId // cannot address candidateId as it changes over the loop
		}
	}
	if addedId == nil {
		return // no single, independent package was added
	}
	added := artifacts[*addedId]
	logging.Debug("Determined that runtime update was triggered by adding package '%s/%s'", added.Namespace, added.Name)

	// Determine the package's direct and indirect dependencies.
	dependencies := buildplan.DependencyMapFor(*addedId, artifacts, filter, showUpdatedPackages)
	directDependencies := make([]artifact.ArtifactID, 0, len(dependencies))
	uniqueDependencies := make(map[artifact.ArtifactID]bool)
	for artifactId, indirectDependencies := range dependencies {
		directDependencies = append(directDependencies, artifactId)
		uniqueDependencies[artifactId] = true
		for _, depId := range indirectDependencies {
			uniqueDependencies[depId] = true
		}
	}
	sort.SliceStable(directDependencies, func(i, j int) bool {
		return artifacts[directDependencies[i]].Name < artifacts[directDependencies[j]].Name
	})
	hasAdditionalIndirectDependencies := len(directDependencies) < len(uniqueDependencies)

	logging.Debug("%s has %d direct dependencies and %d total, unique dependencies", added.Name, len(directDependencies), len(uniqueDependencies))
	if len(directDependencies) == 0 {
		return
	}

	// Process the existing runtime requirements into something we can easily compare against.
	oldRequirements := make(map[string]string)
	for _, source := range filter {
		oldRequirements[fmt.Sprintf("%s/%s", source.Namespace, source.Name)] = *source.Version
	}

	// List additional dependencies.
	out.Notice("") // blank line

	localeKey := "additional_dependencies"
	if hasAdditionalIndirectDependencies {
		localeKey = "additional_total_dependencies"
	}
	version := ""
	if added.Version != nil {
		version = *added.Version
	}
	out.Notice(locale.Tr(localeKey,
		added.Name, version, strconv.Itoa(len(directDependencies)), strconv.Itoa(len(uniqueDependencies))))

	// A direct dependency list item is of the form:
	//   ├─ name@version (X dependencies)
	// or
	//   └─ name@oldVersion → name@newVersion (Updated)
	// depending on whether or not it has subdependencies, and whether or not showUpdatedPackages is
	// `true`.
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
		if numSubs := len(dependencies[dep.ArtifactID]); numSubs > 0 && hasAdditionalIndirectDependencies {
			subdependencies = fmt.Sprintf(" ([ACTIONABLE]%s[/RESET] dependencies)", // intentional leading space
				strconv.Itoa(numSubs))
		}

		item := fmt.Sprintf("[ACTIONABLE]%s@%s[/RESET]%s", // intentional omission of space before last %s
			dep.Name, version, subdependencies)
		if oldVersion, exists := oldRequirements[fmt.Sprintf("%s/%s", dep.Namespace, dep.Name)]; exists && version != "" && oldVersion != version {
			item = fmt.Sprintf("[ACTIONABLE]%s@%s[/RESET] → %s (%s)", dep.Name, oldVersion, item, locale.Tl("updated", "updated"))
		}

		out.Notice(fmt.Sprintf("  [DISABLED]%s[/RESET] %s", prefix, item))
	}

	out.Notice("") // blank line
}

// isDependency iterates over all artifacts and their dependencies in the given changeset, and
// returns whether or not the given artifact is a dependency of any of those artifacts or
// dependencies.
func isDependency(a artifact.ArtifactID, changeset artifact.ArtifactChangeset, artifacts artifact.Map) bool {
	for _, artifactId := range changeset.Added {
		if artifactId == a {
			continue
		}

		for _, depId := range buildplan.RecursiveDependenciesFor(artifactId, artifacts) {
			if a == depId {
				return true
			}
		}
	}

	for _, update := range changeset.Updated {
		for _, depId := range buildplan.RecursiveDependenciesFor(update.ToID, artifacts) {
			if a == depId {
				return true
			}
		}
		for _, depId := range buildplan.RecursiveDependenciesFor(update.FromID, artifacts) {
			if a == depId {
				return true
			}
		}
	}

	return false
}