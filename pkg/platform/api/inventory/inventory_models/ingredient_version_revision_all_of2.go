// Code generated by go-swagger; DO NOT EDIT.

package inventory_models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"strconv"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// IngredientVersionRevisionAllOf2 ingredient version revision all of2
//
// swagger:model ingredientVersionRevisionAllOf2
type IngredientVersionRevisionAllOf2 struct {

	// The build scripts that are used for this revision
	BuildScripts []*BuildScript `json:"build_scripts"`

	// ingredient version id
	// Required: true
	// Format: uuid
	IngredientVersionID *strfmt.UUID `json:"ingredient_version_id"`

	// ingredient version revision id
	// Required: true
	// Format: uuid
	IngredientVersionRevisionID *strfmt.UUID `json:"ingredient_version_revision_id"`

	// links
	// Required: true
	Links *IngredientVersionRevisionAllOf2Links `json:"links"`

	// The patches that apply to this revision
	Patches []*IngredientVersionRevisionPatch `json:"patches"`
}

// Validate validates this ingredient version revision all of2
func (m *IngredientVersionRevisionAllOf2) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateBuildScripts(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateIngredientVersionID(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateIngredientVersionRevisionID(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateLinks(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validatePatches(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *IngredientVersionRevisionAllOf2) validateBuildScripts(formats strfmt.Registry) error {
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

func (m *IngredientVersionRevisionAllOf2) validateIngredientVersionID(formats strfmt.Registry) error {

	if err := validate.Required("ingredient_version_id", "body", m.IngredientVersionID); err != nil {
		return err
	}

	if err := validate.FormatOf("ingredient_version_id", "body", "uuid", m.IngredientVersionID.String(), formats); err != nil {
		return err
	}

	return nil
}

func (m *IngredientVersionRevisionAllOf2) validateIngredientVersionRevisionID(formats strfmt.Registry) error {

	if err := validate.Required("ingredient_version_revision_id", "body", m.IngredientVersionRevisionID); err != nil {
		return err
	}

	if err := validate.FormatOf("ingredient_version_revision_id", "body", "uuid", m.IngredientVersionRevisionID.String(), formats); err != nil {
		return err
	}

	return nil
}

func (m *IngredientVersionRevisionAllOf2) validateLinks(formats strfmt.Registry) error {

	if err := validate.Required("links", "body", m.Links); err != nil {
		return err
	}

	if m.Links != nil {
		if err := m.Links.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("links")
			}
			return err
		}
	}

	return nil
}

func (m *IngredientVersionRevisionAllOf2) validatePatches(formats strfmt.Registry) error {
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

// ContextValidate validate this ingredient version revision all of2 based on the context it is used
func (m *IngredientVersionRevisionAllOf2) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateBuildScripts(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateLinks(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidatePatches(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *IngredientVersionRevisionAllOf2) contextValidateBuildScripts(ctx context.Context, formats strfmt.Registry) error {

	for i := 0; i < len(m.BuildScripts); i++ {

		if m.BuildScripts[i] != nil {
			if err := m.BuildScripts[i].ContextValidate(ctx, formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("build_scripts" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

func (m *IngredientVersionRevisionAllOf2) contextValidateLinks(ctx context.Context, formats strfmt.Registry) error {

	if m.Links != nil {
		if err := m.Links.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("links")
			}
			return err
		}
	}

	return nil
}

func (m *IngredientVersionRevisionAllOf2) contextValidatePatches(ctx context.Context, formats strfmt.Registry) error {

	for i := 0; i < len(m.Patches); i++ {

		if m.Patches[i] != nil {
			if err := m.Patches[i].ContextValidate(ctx, formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("patches" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

// MarshalBinary interface implementation
func (m *IngredientVersionRevisionAllOf2) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *IngredientVersionRevisionAllOf2) UnmarshalBinary(b []byte) error {
	var res IngredientVersionRevisionAllOf2
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
