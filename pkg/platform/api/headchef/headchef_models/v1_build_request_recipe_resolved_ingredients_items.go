// Code generated by go-swagger; DO NOT EDIT.

package headchef_models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"strconv"

	strfmt "github.com/go-openapi/strfmt"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// V1BuildRequestRecipeResolvedIngredientsItems Resolved Ingredient
//
// An ingredient that is part of a recipe's resolved requirements
// swagger:model v1BuildRequestRecipeResolvedIngredientsItems
type V1BuildRequestRecipeResolvedIngredientsItems struct {

	// Alternative ingredient versions which can also satisfy the order's requirement. Each entry in the array is the ID of an ingredient version which could satisfy these requirements.
	Alternatives []strfmt.UUID `json:"alternatives"`

	// An ID to uniquely represent the artifact built from resolved ingredient. The same ingredient version will have different artifact IDs on different platforms, different images, or with different resolved dependencies.
	// Format: uuid
	ArtifactID strfmt.UUID `json:"artifact_id,omitempty"`

	// The custom build scripts for building this ingredient, if any
	BuildScripts []*V1BuildRequestRecipeResolvedIngredientsItemsBuildScriptsItems `json:"build_scripts"`

	// The dependencies in the recipe for this ingredient version. Each item contains an ingredient version UUID which maps to an ingredient version in this recipe.
	Dependencies []*V1BuildRequestRecipeResolvedIngredientsItemsDependenciesItems `json:"dependencies"`

	// ingredient
	// Required: true
	Ingredient *V1BuildRequestRecipeResolvedIngredientsItemsIngredient `json:"ingredient"`

	// The ingredient options of the resolved ingredient which had their conditions met by the recipe
	IngredientOptions []*V1BuildRequestRecipeResolvedIngredientsItemsIngredientOptionsItems `json:"ingredient_options"`

	// ingredient version
	// Required: true
	IngredientVersion *V1BuildRequestRecipeResolvedIngredientsItemsIngredientVersion `json:"ingredient_version"`

	// The patches to apply to this ingredient's source before building, if any
	Patches []*V1BuildRequestRecipeResolvedIngredientsItemsPatchesItems `json:"patches"`

	// The original requirement(s) in the order that were resolved to this ingredient version. This list will be empty if an ingredient was added to the recipe to fulfill a dependency of something else in the order.
	ResolvedRequirements []*V1BuildRequestRecipeResolvedIngredientsItemsResolvedRequirementsItems `json:"resolved_requirements"`
}

// Validate validates this v1 build request recipe resolved ingredients items
func (m *V1BuildRequestRecipeResolvedIngredientsItems) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateAlternatives(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateArtifactID(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateBuildScripts(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateDependencies(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateIngredient(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateIngredientOptions(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateIngredientVersion(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validatePatches(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateResolvedRequirements(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *V1BuildRequestRecipeResolvedIngredientsItems) validateAlternatives(formats strfmt.Registry) error {

	if swag.IsZero(m.Alternatives) { // not required
		return nil
	}

	for i := 0; i < len(m.Alternatives); i++ {

		if err := validate.FormatOf("alternatives"+"."+strconv.Itoa(i), "body", "uuid", m.Alternatives[i].String(), formats); err != nil {
			return err
		}

	}

	return nil
}

func (m *V1BuildRequestRecipeResolvedIngredientsItems) validateArtifactID(formats strfmt.Registry) error {

	if swag.IsZero(m.ArtifactID) { // not required
		return nil
	}

	if err := validate.FormatOf("artifact_id", "body", "uuid", m.ArtifactID.String(), formats); err != nil {
		return err
	}

	return nil
}

func (m *V1BuildRequestRecipeResolvedIngredientsItems) validateBuildScripts(formats strfmt.Registry) error {

	if swag.IsZero(m.BuildScripts) { // not required
		return nil
	}

	for i := 0; i < len(m.BuildScripts); i++ {
		if swag.IsZero(m.BuildScripts[i]) { // not required
			continue
		}

		if m.BuildScripts[i] != nil {
			if err := m.BuildScripts[i].Validate(formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("build_scripts" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

func (m *V1BuildRequestRecipeResolvedIngredientsItems) validateDependencies(formats strfmt.Registry) error {

	if swag.IsZero(m.Dependencies) { // not required
		return nil
	}

	for i := 0; i < len(m.Dependencies); i++ {
		if swag.IsZero(m.Dependencies[i]) { // not required
			continue
		}

		if m.Dependencies[i] != nil {
			if err := m.Dependencies[i].Validate(formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("dependencies" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

func (m *V1BuildRequestRecipeResolvedIngredientsItems) validateIngredient(formats strfmt.Registry) error {

	if err := validate.Required("ingredient", "body", m.Ingredient); err != nil {
		return err
	}

	if m.Ingredient != nil {
		if err := m.Ingredient.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("ingredient")
			}
			return err
		}
	}

	return nil
}

func (m *V1BuildRequestRecipeResolvedIngredientsItems) validateIngredientOptions(formats strfmt.Registry) error {

	if swag.IsZero(m.IngredientOptions) { // not required
		return nil
	}

	for i := 0; i < len(m.IngredientOptions); i++ {
		if swag.IsZero(m.IngredientOptions[i]) { // not required
			continue
		}

		if m.IngredientOptions[i] != nil {
			if err := m.IngredientOptions[i].Validate(formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("ingredient_options" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

func (m *V1BuildRequestRecipeResolvedIngredientsItems) validateIngredientVersion(formats strfmt.Registry) error {

	if err := validate.Required("ingredient_version", "body", m.IngredientVersion); err != nil {
		return err
	}

	if m.IngredientVersion != nil {
		if err := m.IngredientVersion.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("ingredient_version")
			}
			return err
		}
	}

	return nil
}

func (m *V1BuildRequestRecipeResolvedIngredientsItems) validatePatches(formats strfmt.Registry) error {

	if swag.IsZero(m.Patches) { // not required
		return nil
	}

	for i := 0; i < len(m.Patches); i++ {
		if swag.IsZero(m.Patches[i]) { // not required
			continue
		}

		if m.Patches[i] != nil {
			if err := m.Patches[i].Validate(formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("patches" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

func (m *V1BuildRequestRecipeResolvedIngredientsItems) validateResolvedRequirements(formats strfmt.Registry) error {

	if swag.IsZero(m.ResolvedRequirements) { // not required
		return nil
	}

	for i := 0; i < len(m.ResolvedRequirements); i++ {
		if swag.IsZero(m.ResolvedRequirements[i]) { // not required
			continue
		}

		if m.ResolvedRequirements[i] != nil {
			if err := m.ResolvedRequirements[i].Validate(formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("resolved_requirements" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

// MarshalBinary interface implementation
func (m *V1BuildRequestRecipeResolvedIngredientsItems) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *V1BuildRequestRecipeResolvedIngredientsItems) UnmarshalBinary(b []byte) error {
	var res V1BuildRequestRecipeResolvedIngredientsItems
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
