package requirements

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildplan"
	"github.com/ActiveState/cli/pkg/platform/runtime/store"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
)

// maxListLength is the maximum number of direct dependencies to show before adding a "more..."
// sentinel.
const maxListLength = 10

// showUpdatedPackages specifies whether or not to include updated dependencies in the direct
// dependencies list, and whether or not to include updated dependencies when calculating indirect
// dependency numbers.
const showUpdatedPackages = true

// outputAdditionalRequirements computes and lists the additional dependencies being installed for
// the given package name.
// This should only be called if a package or bundle is being added or updated. Otherwise, the
// results may be nonsensical.
func (r *RequirementOperation) outputAdditionalRequirements(parentCommitId, commitId strfmt.UUID, packageName string) (rerr error) {
	pg := output.StartSpinner(r.Output, locale.T("progress_dependencies"), constants.TerminalAnimationInterval)
	defer func() {
		if rerr == nil {
			return
		}
		pg.Stop(locale.T("progress_fail"))
	}()
	bp := model.NewBuildPlannerModel(r.Auth)

	// Fetch old build plan to compare against.
	tgt := target.NewProjectTarget(r.Project, &parentCommitId, target.TriggerPackage)
	oldBuildPlan, err := store.New(tgt.Dir()).BuildPlan()
	if err != nil {
		if errs.Matches(err, store.ErrNoBuildPlanFile) {
			var oldBuildResult *model.BuildResult
			oldBuildResult, err = bp.FetchBuildResult(parentCommitId, r.Project.Owner(), r.Project.Name())
			oldBuildPlan = oldBuildResult.Build
		}
		if err != nil {
			return errs.Wrap(err, "Unable to fetch previous build plan to compare against")
		}
	}

	// Process old build plan's requirements into something we can easily compare against.
	oldRequirements := make(map[string]string)
	for _, source := range oldBuildPlan.Sources {
		oldRequirements[fmt.Sprintf("%s/%s", source.Namespace, source.Name)] = source.Version
	}

	// Fetch new build plan to compare with.
	// Note: ideally we would wait until this call is made during runtime setup to perform this
	// additional requirements computation, but the runtime would have to know that a particular
	// package was just added/updated, and communicating that information from this package would
	// be extremely difficult.
	// Even though this is an initially expensive API call (i.e. the buildplanner must perform the
	// solve and report the results), subsequent calls will be fast, returning a cached result.
	// Therefore, it's not worth the added plumbing/communication effort to avoid one additional
	// API call from the runtime for a cached buildplan result.
	newBuildResult, err := bp.FetchBuildResult(commitId, r.Project.Owner(), r.Project.Name())
	if err != nil {
		return errs.Wrap(err, "Unable to fetch new build plan to compare with")
	}

	// Find the resolved version of the newly added or updated package.
	packageVersion := locale.T("constraint_auto")
	for _, source := range newBuildResult.Build.Sources {
		if source.Name == packageName &&
			(model.NamespaceMatch(source.Namespace, model.NamespaceBundlesMatch) ||
				model.NamespaceMatch(source.Namespace, model.NamespacePackageMatch)) {
			packageVersion = source.Version
			break
		}
	}

	// Process new build plan's requirements into something we can easily compare with.
	newArtifacts, err := buildplan.NewArtifactListing(newBuildResult.Build, false, r.Config)
	if err != nil {
		return errs.Wrap(err, "Unable to create artifact listing for new build plan")
	}
	newArtifactMap, err := newArtifacts.RuntimeClosure()
	if err != nil {
		return errs.Wrap(err, "Unable to compute runtime closure for new build plan")
	}

	// Determine the package's direct and indirect dependencies.
	var directDependencies []*artifact.Artifact
	numDependencies := make(map[artifact.ArtifactID]int)
	totalDependencies := 0
	for _, artf := range newArtifactMap {
		if artf.Name != packageName {
			continue
		}
		directDependencies = computeDependencies(&artf, &newArtifactMap, false)
		directDependencies = filterDependencies(directDependencies, &oldRequirements, showUpdatedPackages)
		totalDependencies = len(directDependencies)
		for _, dep := range directDependencies {
			indirectDependencies := computeDependencies(dep, &newArtifactMap, true)
			indirectDependencies = filterDependencies(indirectDependencies, &oldRequirements, showUpdatedPackages)
			numDependencies[dep.ArtifactID] = len(indirectDependencies)
			totalDependencies += len(indirectDependencies)
		}
		break
	}

	pg.Stop(locale.T("progress_success"))
	if len(directDependencies) == 0 {
		return nil
	}

	// List additional dependencies.
	r.Output.Notice("") // blank line

	localeKey := "additional_dependencies"
	if len(directDependencies) < totalDependencies {
		localeKey = "additional_total_dependencies"
	}
	r.Output.Notice(locale.Tr(localeKey,
		packageName, packageVersion, strconv.Itoa(len(directDependencies)), strconv.Itoa(totalDependencies)))

	// A direct dependency list item is of the form:
	//   ├─ name@version (X dependencies)
	// or
	//   └─ name@oldVersion → name@newVersion (Updated)
	// depending on whether or not it has subdependencies, and whether or not showUpdatedPackages is
	// `true`.
	for i, dep := range directDependencies {
		if i > maxListLength {
			break
		}
		prefix := "├─"
		if i == len(directDependencies)-1 || i == maxListLength {
			prefix = "└─"
		}

		version := ""
		if dep.Version != nil {
			version = *dep.Version
		}

		subdependencies := ""
		if numSubs := numDependencies[dep.ArtifactID]; numSubs > 0 {
			subdependencies = fmt.Sprintf(" ([ACTIONABLE]%s[/RESET] dependencies)", strconv.Itoa(numSubs)) // intentional leading space
		}

		item := fmt.Sprintf("[ACTIONABLE]%s@%s[/RESET]%s", dep.Name, version, subdependencies) // intentional omission of space before last %s
		if oldVersion, exists := oldRequirements[fmt.Sprintf("%s/%s", dep.Namespace, dep.Name)]; exists && version != "" && oldVersion != version {
			item = fmt.Sprintf("[ACTIONABLE]%s@%s[/RESET] → %s (%s)", dep.Name, oldVersion, item, locale.Tl("updated", "Updated"))
		}

		if i == maxListLength && i < len(directDependencies)-1 {
			item = locale.Tl("more_dependencies", "{{.V0}} more...", strconv.Itoa(len(directDependencies)-1-i))
		}

		r.Output.Notice(fmt.Sprintf("  [DISABLED]%s[/RESET] %s", prefix, item))
	}

	r.Output.Notice("") // blank line

	return nil
}

// computeDependencies returns a sorted, unique list of dependencies for the given artifact.
// `artifactMap` is a map of artifact IDs to full artifact structs because artifact dependency lists
// only contain artifact IDs.
// When `indirect` is true, also includes dependencies of the given artifact's dependencies. If
// there are duplicate subdependencies, only one of them is kept.
func computeDependencies(artf *artifact.Artifact, artifactMap *artifact.Map, indirect bool) []*artifact.Artifact {
	dependencies := make(map[artifact.ArtifactID]*artifact.Artifact)
	for _, artifactId := range artf.Dependencies {
		dep := (*artifactMap)[artifactId]
		dependencies[dep.ArtifactID] = &dep
		if indirect {
			for _, idep := range computeDependencies(&dep, artifactMap, indirect) {
				dependencies[idep.ArtifactID] = idep
			}
		}
	}

	list := make([]*artifact.Artifact, 0)
	for _, dep := range dependencies {
		list = append(list, dep)
	}
	sort.SliceStable(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})
	return list
}

// filterDependencies removes from the given dependency list any dependencies seen in the given
// existing dependency map.
// When `keepUpdated` is true, dependencies that changed version (i.e. they were updated updated)
// are not filtered out.
func filterDependencies(dependencies []*artifact.Artifact, existingArtifacts *map[string]string, keepUpdated bool) []*artifact.Artifact {
	added := make([]*artifact.Artifact, 0)
	for _, dep := range dependencies {
		key := fmt.Sprintf("%s/%s", dep.Namespace, dep.Name)
		oldVersion, exists := (*existingArtifacts)[key]
		if !exists || (keepUpdated && dep.Version != nil && oldVersion != *dep.Version) {
			added = append(added, dep)
		}
	}
	return added
}
