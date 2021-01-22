package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	uuid "github.com/satori/go.uuid"
)

// IngredientMetadata captures all the fields of a resolved ingredient that the
// wrapper needs in order to execute a build of the ingredient.
type IngredientMetadata struct {
	IngredientID        uuid.UUID `json:"ingredient_id"`
	Namespace           string    `json:"namespace"`
	Name                string    `json:"name"`
	IngredientVersionID uuid.UUID `json:"ingredient_version_id"`
	Version             string    `json:"version"`
	Revision            uint      `json:"revision"`
	SourceURI           string    `json:"source_uri"`
	SourceChecksum      string    `json:"source_checksum"`
	Options             []string  `json:"options"`
}

// NewIngredientMetadata instantiates and returns a new *IngredientMetadata and
// populates it based on a resolved ingredient from a recipe.
func NewIngredientMetadata(
	ingredient *inventory_models.ResolvedIngredient,
) *IngredientMetadata {
	options := []string{}
	for _, ingredientOption := range ingredient.IngredientOptions {
		options = append(options, ingredientOption.CommandLineArgs...)
	}

	return &IngredientMetadata{
		IngredientID:        uuid.Must(uuid.FromString(ingredient.Ingredient.IngredientID.String())),
		Namespace:           *ingredient.Ingredient.PrimaryNamespace,
		Name:                *ingredient.Ingredient.Name,
		IngredientVersionID: uuid.Must(uuid.FromString(ingredient.IngredientVersion.IngredientVersionID.String())),
		Version:             *ingredient.IngredientVersion.Version,
		Revision:            uint(*ingredient.IngredientVersion.RevisionedResource.Revision),
		SourceURI:           ingredient.IngredientVersion.PlatformSourceURI.String(),
		SourceChecksum:      *ingredient.IngredientVersion.SourceChecksum,
		Options:             options,
	}
}

// FullName is a human-readable name describing this ingredient. It is unique
// amongst all ingredients, but doesn't uniquely identify an artifact since it
// doesn't take the build environment/dependencies into account.
func (im *IngredientMetadata) FullName() string {
	return strings.Join([]string{
		im.Namespace,
		im.Name,
		im.Version,
		strconv.FormatInt(int64(im.Revision), 10),
	}, "|")
}

// IsSameIngredientVersionRevision returns true if this `*IngredientMetadata`
// represents the exact same ingredient version revision as the provided
// `*IngredientMetadata`, otherwise false.
func (im *IngredientMetadata) IsSameIngredientVersionRevision(other *IngredientMetadata) bool {
	return im.IngredientID == other.IngredientID &&
		im.IngredientVersionID == other.IngredientVersionID &&
		im.Revision == other.Revision
}

// JSONMetadataScanner is a wrapper around an IngredientMetadata for
// initializing its value from an ingredient_metadata record stored in the DB.
// It wraps a double pointer to an IngredientMetadata value so that it can
// initialize the value to nil should the DB record be NULL.
//
// Note that this scanner requires the DB record to be formatted as a JSON
// object, which can be done by including `TO_JSON(<column_name>)` in the SELECT
// query. The reason being that JSON can be more easily/accurately decoded into
// a Go struct than the default serialization format that Postgres uses for
// composite types, which the ingredient_metadata DB type is.
type JSONMetadataScanner struct {
	target **IngredientMetadata
}

// NewJSONMetadataScanner creates a new scanner
func NewJSONMetadataScanner(target **IngredientMetadata) *JSONMetadataScanner {
	return &JSONMetadataScanner{
		target: target,
	}
}

// Scan implements the sql.Scanner interface
func (s *JSONMetadataScanner) Scan(value interface{}) error {
	if value == nil {
		*s.target = nil
		return nil
	}

	bytesValue, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("Cannot convert value of type %T to an ingredient metadata", value)
	}

	decoder := json.NewDecoder(bytes.NewReader(bytesValue))
	decoder.DisallowUnknownFields()

	scanTarget := &IngredientMetadata{}
	err := decoder.Decode(scanTarget)
	if err != nil {
		return fmt.Errorf("Unable to parse value as JSON representation of ingredient metadata: %s", err.Error())
	}

	*s.target = scanTarget
	return nil
}

// IngredientMetadataSorter can be passed to sort.SliceStable to sort a slice of
// IngredientMetadata by the namespace/name/version/revision of the metadata.
func IngredientMetadataSorter(ims []*IngredientMetadata) func(int, int) bool {
	return func(i, j int) bool {
		return ims[i].FullName() < ims[j].FullName()
	}
}
