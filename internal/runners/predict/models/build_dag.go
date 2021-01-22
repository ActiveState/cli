package models

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/go-openapi/strfmt"
	uuid "github.com/satori/go.uuid"
)

// InvalidRecipeErrorType is set on the ErrInvalidRecipe error to more
// granularly indicate how the recipe was invalid
type InvalidRecipeErrorType int

const (
	_ InvalidRecipeErrorType = iota
	// ErrDependencyCycle is if an artifact is found to contain itself in its
	// own builder closure
	ErrDependencyCycle
	// ErrDisallowedDependency is if a builder has non-runtime dependency or a
	// dependency on something not in the builder of builder-lib namespace
	ErrDisallowedDependency
	// ErrInvalidPatchURI is if an artifact has a patch URI that is not a valid
	// S3 URI
	ErrInvalidPatchURI
	// ErrMissingArtifactID is if an artifact doesn't have an artifact ID
	ErrMissingArtifactID
	// ErrMissingBuilder is if an artifact doesn't have a builder as a
	// dependency
	ErrMissingBuilder
	// ErrMissingDependency is if an artifact references a dependency that
	// doesn't appear in the recipe
	ErrMissingDependency
	// ErrMultipleBuilders is if an artifact has more than one builder as a
	// recipe
	ErrMultipleBuilders
)

// ErrInvalidRecipe is returned by `ParseRecipe` when the provided recipe cannot
// be parsed into a build DAG for an alternative build. This should be
// considered a validation error.
type ErrInvalidRecipe struct {
	ArtifactID uuid.UUID
	Type       InvalidRecipeErrorType
	message    string
}

func (e *ErrInvalidRecipe) Error() string {
	return e.message
}

func newInvalidRecipeError(
	errType InvalidRecipeErrorType,
	artifactID uuid.UUID,
	ingr *IngredientMetadata,
	msg string,
) *ErrInvalidRecipe {
	return &ErrInvalidRecipe{
		ArtifactID: artifactID,
		Type:       errType,
		message:    fmt.Sprintf("Resolved ingredient %s %s", ingr.FullName(), msg),
	}
}

// RecipeBuildDAG wraps a DAG of *Artifact objects representing the build of a
// particular recipe. It all provides several APIs for manipulating/traversing
// the DAG.
type RecipeBuildDAG struct {
	RecipeID         uuid.UUID
	TerminalArtifact *Artifact
}

// A handler function that will check if this type of error should be considered
// an error for this recipe parsing and, if so, returns that error
type missingDepHandler func(artifactID uuid.UUID, ingr *IngredientMetadata, missingDepVersionID strfmt.UUID) error

// A link in a dependency chain. Needed to track cycles in case they occur
type depPathSegment struct {
	Ingredient *inventory_models.ResolvedIngredient
	Dependency *inventory_models.ResolvedIngredientDependency
}

func (s depPathSegment) String() string {
	depTypeStrings := make([]string, 0, len(s.Dependency.DependencyTypes))
	for _, depType := range s.Dependency.DependencyTypes {
		depTypeStrings = append(depTypeStrings, string(depType))
	}

	return fmt.Sprintf(
		"%s -[%s]>",
		NewIngredientMetadata(s.Ingredient).FullName(),
		strings.Join(depTypeStrings, ","),
	)
}

func makeSubpath(
	depPath []depPathSegment,
	ingredient *inventory_models.ResolvedIngredient,
	dependency *inventory_models.ResolvedIngredientDependency,
) []depPathSegment {
	subPath := make([]depPathSegment, len(depPath))
	copy(subPath, depPath)

	subPath = append(
		subPath,
		depPathSegment{
			Ingredient: ingredient,
			Dependency: dependency,
		},
	)

	return subPath
}

// ParseRecipe takes a recipe from the solver and constructs a build DAG out of
// it. The nodes of the DAG represent individual artifacts to be built and the
// edges represent build dependencies between these artifacts. Each artifact is
// given an artifact ID generated using a merkle tree-like approach, using the
// artifacts' properties and the sub-DAG below it as input.
func ParseRecipe(recipe *inventory_models.Recipe) (*RecipeBuildDAG, error) {
	return parseRecipe(recipe, func(
		artifactID uuid.UUID,
		ingr *IngredientMetadata,
		missingDepVersionID strfmt.UUID,
	) error {
		return newInvalidRecipeError(
			ErrMissingDependency,
			artifactID,
			ingr,
			fmt.Sprintf(
				"depends on ingredient version %s, which isn't in the recipe",
				missingDepVersionID.String(),
			),
		)
	})
}

func parseRecipe(recipe *inventory_models.Recipe, handleMissingDep missingDepHandler) (*RecipeBuildDAG, error) {
	ingredientsByVersionID := map[strfmt.UUID]*inventory_models.ResolvedIngredient{}
	artifactsByVersionID := map[strfmt.UUID]*Artifact{}
	var err error

	for _, resolvedIngredient := range recipe.ResolvedIngredients {
		versionID := *resolvedIngredient.IngredientVersion.IngredientVersionID
		ingredientsByVersionID[versionID] = resolvedIngredient

		if isBuilderRelatedNamespace(*resolvedIngredient.Ingredient.PrimaryNamespace) {
			// Builder ingredients and their libraries don't get their own
			// artifacts in the DAG since they don't themselves get built, just
			// used in the build of non-builder ingredients.
			continue
		}

		artifactsByVersionID[versionID], err = makeBaseIngredientArtifact(recipe.Platform, recipe.Image, resolvedIngredient)
		if err != nil {
			return nil, err
		}
	}

	for versionID, artifact := range artifactsByVersionID {
		ingredient := ingredientsByVersionID[versionID]

		err = resolveArtifactDependencies(
			handleMissingDep,
			artifactsByVersionID,
			ingredientsByVersionID,
			artifact,
			ingredient,
		)
		if err != nil {
			return nil, err
		}

		for _, patch := range ingredient.Patches {
			// Despite being called 'content', the field is used to store the
			// URI of the patch in S3.
			patchURI := *patch.Content
			// XXX: We eventually want standarize all patches to use a full S3
			// URI, but to support an expeditious implementation of hybrid
			// builds, we're relaxing this constraint specifically for
			// ingredients builds by camel since it _can_ resolve these non-S3
			// URIs. This exception case should be taken out when hybrid builds
			// are decommissioned.
			if artifact.Builder.Name != "camel" && !strings.HasPrefix(patchURI, "s3://") {
				return nil, newInvalidRecipeError(
					ErrInvalidPatchURI,
					artifact.ArtifactID,
					artifact.Ingredient,
					fmt.Sprintf(
						"has a patch %d with an invalid S3 URI: '%s'",
						*patch.SequenceNumber,
						patchURI,
					),
				)
			}

			artifact.PatchURIs = append(artifact.PatchURIs, *patch.Content)
		}
	}

	recipeID := uuid.Must(uuid.FromString(recipe.RecipeID.String()))
	return &RecipeBuildDAG{
		RecipeID:         recipeID,
		TerminalArtifact: makeTerminalArtifact(artifactsByVersionID, recipeID, recipe.Platform, recipe.Image),
	}, nil
}

func isBuilderRelatedNamespace(namespace string) bool {
	return namespace == "builder" || namespace == "builder-lib"
}

func makeBaseIngredientArtifact(
	platform *inventory_models.Platform,
	image *inventory_models.Image,
	ingredient *inventory_models.ResolvedIngredient,
) (*Artifact, error) {
	ingredientMetadata := NewIngredientMetadata(ingredient)

	artifactID, err := uuid.FromString(ingredient.ArtifactID.String())
	if err != nil {
		return nil, newInvalidRecipeError(
			ErrMissingArtifactID,
			uuid.Nil,
			ingredientMetadata,
			"does not have a valid artifact ID",
		)
	}

	return &Artifact{
		ArtifactID:   artifactID,
		BuildType:    Builder,
		Ingredient:   ingredientMetadata,
		ImageType:    *image.Type,
		ImageName:    *image.Name,
		ImageVersion: *image.Version,
		PlatformName: *platform.DisplayName,
		PlatformID:   uuid.Must(uuid.FromString(platform.PlatformID.String())),
		LastModified: DBFriendlyTimestamp(),
	}, nil
}

func resolveArtifactDependencies(
	handleMissingDep missingDepHandler,
	artifactsByVersionID map[strfmt.UUID]*Artifact,
	ingredientsByVersionID map[strfmt.UUID]*inventory_models.ResolvedIngredient,
	artifact *Artifact,
	ingredient *inventory_models.ResolvedIngredient,
) error {
	buildDependencyClosure := map[strfmt.UUID]*Artifact{}

	for _, dependency := range ingredient.Dependencies {
		if !isDependencyType(dependency, "build") {
			// An artifact's non-build dependencies don't matter for building
			// the artifact itself. They may come into play if the artifact is
			// used as a dependency or is tested.
			continue
		}

		depVersionID := *dependency.IngredientVersionID
		if dependencyArtifact, ok := artifactsByVersionID[depVersionID]; ok {
			buildDependencyClosure[depVersionID] = dependencyArtifact

			err := resolveTransitiveRuntimeDependencies(
				handleMissingDep,
				artifactsByVersionID,
				ingredientsByVersionID,
				buildDependencyClosure,
				makeSubpath([]depPathSegment{}, ingredient, dependency),
				ingredientsByVersionID[depVersionID],
			)
			if err != nil {
				return err
			}
		} else if builder, ok := ingredientsByVersionID[depVersionID]; ok {
			// Assuming it's a builder because it would have been in
			// `artifactsByVersionID` if it wasn't

			if artifact.Builder != nil {
				return newInvalidRecipeError(
					ErrMultipleBuilders,
					artifact.ArtifactID,
					artifact.Ingredient,
					fmt.Sprintf(
						"has multiple builders: %s and %s",
						artifact.Builder.FullName(),
						NewIngredientMetadata(builder).FullName(),
					),
				)
			}

			artifact.Builder = NewIngredientMetadata(builder)

			builderDependencyClosure := map[strfmt.UUID]*BuilderDependency{}
			err := resolveBuilderDependencies(
				handleMissingDep,
				ingredientsByVersionID,
				builderDependencyClosure,
				builder,
			)
			if err != nil {
				return err
			}

			for _, builderDependency := range builderDependencyClosure {
				artifact.BuilderDependencies = append(artifact.BuilderDependencies, builderDependency)
			}
			// Sort for the sake of producing a consistent artifact ID for a given
			// set of dependencies.
			sort.SliceStable(
				artifact.BuilderDependencies,
				BuilderDependencySorter(artifact.BuilderDependencies),
			)
		} else {
			err := handleMissingDep(artifact.ArtifactID, artifact.Ingredient, depVersionID)
			if err != nil {
				return err
			}
		}
	}

	if artifact.Builder == nil {
		return newInvalidRecipeError(
			ErrMissingBuilder,
			artifact.ArtifactID,
			artifact.Ingredient,
			"has no builder in the recipe",
		)
	}

	for _, buildDependency := range buildDependencyClosure {
		artifact.BuildDependencies = append(artifact.BuildDependencies, buildDependency)
	}
	// Sort for the sake of producing a consistent artifact ID for a given
	// set of dependencies.
	sort.SliceStable(
		artifact.BuildDependencies,
		ArtifactSorter(artifact.BuildDependencies),
	)

	return nil
}

func resolveTransitiveRuntimeDependencies(
	handleMissingDep missingDepHandler,
	artifactsByVersionID map[strfmt.UUID]*Artifact,
	ingredientsByVersionID map[strfmt.UUID]*inventory_models.ResolvedIngredient,
	buildDependencyClosure map[strfmt.UUID]*Artifact,
	depPath []depPathSegment,
	dependency *inventory_models.ResolvedIngredient,
) error {
	for _, transitiveDep := range dependency.Dependencies {
		if !isDependencyType(transitiveDep, "runtime") {
			// Only the transitive runtime dependencies of an artifact's build
			// dependencies are needed to build the artifact.
			continue
		}

		depVersionID := *transitiveDep.IngredientVersionID
		if _, ok := buildDependencyClosure[depVersionID]; ok {
			// Dependency is already in the closure, no need to resolved its
			// dependencies again
			continue
		}

		subpath := makeSubpath(depPath, dependency, transitiveDep)
		transitiveDepArtifact, ok := artifactsByVersionID[depVersionID]
		if !ok {
			depArtifact := artifactsByVersionID[*dependency.IngredientVersion.IngredientVersionID]
			err := handleMissingDep(depArtifact.ArtifactID, NewIngredientMetadata(dependency), depVersionID)
			if err != nil {
				return err
			}
			continue
		} else if subpath[0].Ingredient.ArtifactID.String() == transitiveDepArtifact.ArtifactID.String() {
			pathSegmentStrings := make([]string, 0, len(subpath))
			for _, segment := range subpath {
				pathSegmentStrings = append(pathSegmentStrings, segment.String())
			}

			return newInvalidRecipeError(
				ErrDependencyCycle,
				transitiveDepArtifact.ArtifactID,
				transitiveDepArtifact.Ingredient,
				fmt.Sprintf(
					"depends upon itself to build: %s %s",
					strings.Join(pathSegmentStrings, " "),
					transitiveDepArtifact.Ingredient.FullName(),
				),
			)
		}

		buildDependencyClosure[depVersionID] = transitiveDepArtifact

		// All transitive runtime dependencies of a build dependency must be
		// included in an artifact's `BuildDependencies` list, since they all
		// must be finished building and then installed by the build wrapper for
		// the artifact's build to work.
		err := resolveTransitiveRuntimeDependencies(
			handleMissingDep,
			artifactsByVersionID,
			ingredientsByVersionID,
			buildDependencyClosure,
			subpath,
			ingredientsByVersionID[depVersionID],
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func resolveBuilderDependencies(
	handleMissingDep missingDepHandler,
	ingredientsByVersionID map[strfmt.UUID]*inventory_models.ResolvedIngredient,
	builderDependencyClosure map[strfmt.UUID]*BuilderDependency,
	builder *inventory_models.ResolvedIngredient,
) error {
	for _, builderDep := range builder.Dependencies {
		if !isDependencyType(builderDep, "runtime") {
			return newInvalidRecipeError(
				ErrDisallowedDependency,
				uuid.FromStringOrNil(builder.ArtifactID.String()),
				NewIngredientMetadata(builder),
				fmt.Sprintf(
					"has a non-runtime dependency on ingredient version %s. Only runtime dependencies are supported for builders.",
					builderDep.IngredientVersionID.String(),
				),
			)
		}

		depVersionID := *builderDep.IngredientVersionID
		if _, ok := builderDependencyClosure[depVersionID]; ok {
			// Dependency is already in the closure, no need to resolved its
			// dependencies again
			continue
		}

		dependencyIngredient, ok := ingredientsByVersionID[depVersionID]
		if !ok {
			err := handleMissingDep(
				uuid.FromStringOrNil(builder.ArtifactID.String()),
				NewIngredientMetadata(builder),
				depVersionID,
			)
			if err != nil {
				return err
			}
			continue
		}

		if !isBuilderRelatedNamespace(*dependencyIngredient.Ingredient.PrimaryNamespace) {
			return newInvalidRecipeError(
				ErrDisallowedDependency,
				uuid.FromStringOrNil(builder.ArtifactID.String()),
				NewIngredientMetadata(builder),
				fmt.Sprintf(
					"has dependency on %s. Only ingredients in the builder or builder-lib namespaces are supported as dependencies for builders.",
					NewIngredientMetadata(dependencyIngredient).FullName(),
				),
			)
		}

		builderDependencyClosure[depVersionID] = NewBuilderDependency(dependencyIngredient)

		// All transitive runtime dependencies of a builder must be included in
		// an artifact's `BuilderDependencies` list, since they all must be
		// installed by the build wrapper for the builder to work.
		err := resolveBuilderDependencies(
			handleMissingDep,
			ingredientsByVersionID,
			builderDependencyClosure,
			dependencyIngredient,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func isDependencyType(
	dep *inventory_models.ResolvedIngredientDependency,
	targetType string,
) bool {
	for _, dependencyType := range dep.DependencyTypes {
		if string(dependencyType) == targetType {
			return true
		}
	}
	return false
}

func makeTerminalArtifact(
	artifactsByVersionID map[strfmt.UUID]*Artifact,
	recipeID uuid.UUID,
	platform *inventory_models.Platform,
	image *inventory_models.Image,
) *Artifact {
	// XXX: This is a placeholder for an eventual build step that will package
	// all the built artifact into single distributable. Currently it does
	// nothing and just serves the purpose of being a singular root node for the
	// build DAG.
	terminalArtifact := &Artifact{
		// XXX: Since artifact IDs are calculated in the solver now and this
		// artifact is not represented in the recipe, there is no artifact ID
		// generated for this artifact. As a stopgap for now, we are just using
		// the recipe ID as the artifact ID since there will always be exactly
		// one of these artifacts per recipe. In the future, when build DAGs get
		// more complex and may have multiple non-builder build steps, we'll
		// need to figure out a better approach.
		ArtifactID: recipeID,
		BuildType:  Packager,
		Builder: &IngredientMetadata{
			Namespace:      "builder",
			Name:           "noop-builder",
			Version:        "0.0.1",
			Revision:       1,
			SourceURI:      "s3://platform-sources/builder/21d227eee2d263e171e45ab7357140220174ca83691fab65e0d422eee44e609f/noop-builder.tar.gz",
			SourceChecksum: "21d227eee2d263e171e45ab7357140220174ca83691fab65e0d422eee44e609f",
			Options:        []string{},
		},
		ImageType:           *image.Type,
		ImageName:           *image.Name,
		ImageVersion:        *image.Version,
		PlatformName:        *platform.DisplayName,
		PlatformID:          uuid.Must(uuid.FromString(platform.PlatformID.String())),
		LastModified:        DBFriendlyTimestamp(),
		PatchURIs:           []string{},
		BuilderDependencies: []*BuilderDependency{},
	}

	// For the purposes of this placeholder, assume this packaging artifact
	// depends on all ingredient artifacts. That way it only runs after
	// everything else has built.
	for _, artifact := range artifactsByVersionID {
		terminalArtifact.BuildDependencies = append(
			terminalArtifact.BuildDependencies,
			artifact,
		)
	}

	// Sort for the sake of producing a consistent artifact ID for a given
	// set of dependencies.
	sort.SliceStable(
		terminalArtifact.BuildDependencies,
		ArtifactSorter(terminalArtifact.BuildDependencies),
	)

	return terminalArtifact
}

// PostOrder returns a list of all the nodes in this build DAG post-ordered.
// This means that for two artifacts in the DAG A and B, if there is a
// dependency relationship (direct or transitive) of A on B, then B will appear
// before A in the returned list.
func (r *RecipeBuildDAG) PostOrder() []*Artifact {
	visited := map[uuid.UUID]bool{}
	order := []*Artifact{}

	r.TerminalArtifact.postOrder(visited, &order)

	return order
}

// InitializeBuildState walks through the all the artifacts in the build DAG and
// initializes their build states to the appropriate value. If any artifacts are
// contained in the passed map of existing artifacts, their build state is
// initialized to that existing value. Otherwise, each artifact's build state is
// initialized to:
//  * READY if it has no dependencies or if all its dependencies are in the
//    SUCCEEDED state. It additionally initializes the ReadyAt field to the
//    current timestamp.
//  * BLOCKED if it has dependencies and none of its dependencies are in a
//    failed state.
//  * DOOMED if has at least one dependency in a failed state
func (r *RecipeBuildDAG) InitializeBuildState(existingArtifacts map[uuid.UUID]*Artifact) {
	r.TerminalArtifact.initializeBuildState(existingArtifacts)
}

func (r *RecipeBuildDAG) String() string {
	b := &strings.Builder{}
	b.WriteString(r.TerminalArtifact.String())
	b.WriteRune('\n')

	buildString(b, map[uuid.UUID]bool{}, "", r.TerminalArtifact)

	return b.String()
}

func buildString(
	b *strings.Builder,
	visited map[uuid.UUID]bool,
	prefix string,
	artifact *Artifact,
) {
	if _, ok := visited[artifact.ArtifactID]; ok {
		if len(artifact.BuildDependencies) > 0 {
			b.WriteString(prefix)
			b.WriteString("    [dependencies already printed]\n")
		}
		return
	}
	visited[artifact.ArtifactID] = true

	for i, dep := range artifact.BuildDependencies {
		b.WriteString(prefix)

		childPrefix := prefix
		if i == len(artifact.BuildDependencies)-1 {
			b.WriteString("└── ")
			childPrefix += "    "
		} else {
			b.WriteString("├── ")
			childPrefix += "│   "
		}

		b.WriteString(dep.String())
		b.WriteRune('\n')
		buildString(b, visited, childPrefix, dep)
	}
}
