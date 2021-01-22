package models

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	uuid "github.com/satori/go.uuid"
)

// BuilderDependency represents all the fields of an ingredient in the
// `builder-lib` namespace which are needed to be able to pull the ingredient in
// as a builder's dependency. Since we are treating ingredients in this
// namespace as "pre-built", the platform source URI field is assumed to contain
// the URI of a tarball resembling a built artifact rather than a source tree.
type BuilderDependency struct {
	ArtifactID uuid.UUID `json:"artifact_id"`
	URI        string    `json:"uri"`
	Checksum   string    `json:"checksum"`
}

// NewBuilderDependency instantiates and returns a new *BuilderDependency and
// populates it based on a resolved ingredient from a recipe.
func NewBuilderDependency(
	ingredient *inventory_models.ResolvedIngredient,
) *BuilderDependency {
	return &BuilderDependency{
		ArtifactID: uuid.Must(uuid.FromString(ingredient.ArtifactID.String())),
		URI:        ingredient.IngredientVersion.PlatformSourceURI.String(),
		Checksum:   *ingredient.IngredientVersion.SourceChecksum,
	}
}

// JSONBuilderDependencyArrayScanner is a wrapper around an BuilderDependency
// slice for initializing its value from an array of artifact_builder_dependency
// records stored in the DB. It wraps a pointer to an BuilderDependency slice
// value so that it can initialize the value to a nil slice should the DB record
// be NULL.
//
// Note that this scanner requires the DB record to be formatted as a JSON
// array, which can be done by including `TO_JSON(<column_name>)` in the SELECT
// query. The reason being that JSON can be more easily/accurately decoded into
// a Go struct than the default serialization format that Postgres uses for
// composite types, which the artifact_builder_dependency DB type is.
type JSONBuilderDependencyArrayScanner struct {
	target *[]*BuilderDependency
}

// NewJSONBuilderDependencyArrayScanner creates a new scanner
func NewJSONBuilderDependencyArrayScanner(target *[]*BuilderDependency) *JSONBuilderDependencyArrayScanner {
	return &JSONBuilderDependencyArrayScanner{
		target: target,
	}
}

// Scan implements the sql.Scanner interface
func (s *JSONBuilderDependencyArrayScanner) Scan(value interface{}) error {
	if value == nil {
		*s.target = nil
		return nil
	}

	bytesValue, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("Cannot convert value of type %T to a builder dependency slice", value)
	}

	decoder := json.NewDecoder(bytes.NewReader(bytesValue))
	decoder.DisallowUnknownFields()
	err := decoder.Decode(s.target)

	if err != nil {
		// Apparently when unmarshalling fails with an unknown field error, the
		// json lib leaves the variable it's unmarshalling into in some invalid,
		// partially-unmarshalled state. So we gotta clean it up ourselves.
		*s.target = nil
		return fmt.Errorf("Unable to parse value as JSON representation of builder dependency slice: %s", err.Error())
	}

	return nil
}

// BuilderDependencySorter can be passed to sort.SliceStable to sort a slice of
// BuilderDependency by its URI, which seemed like the least bad field to use to
// get a consistent sort order.
func BuilderDependencySorter(ims []*BuilderDependency) func(int, int) bool {
	return func(i, j int) bool {
		return ims[i].URI < ims[j].URI
	}
}
