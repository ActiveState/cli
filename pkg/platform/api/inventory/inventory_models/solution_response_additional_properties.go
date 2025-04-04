// Code generated by go-swagger; DO NOT EDIT.

package inventory_models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// SolutionResponseAdditionalProperties solution response additional properties
//
// swagger:model solutionResponseAdditionalProperties
type SolutionResponseAdditionalProperties struct {

	// The location of the recipe
	// Required: true
	// Format: uri
	Link *strfmt.URI `json:"link"`

	// recipe id
	// Required: true
	// Format: uuid
	RecipeID *strfmt.UUID `json:"recipe_id"`
}

// Validate validates this solution response additional properties
func (m *SolutionResponseAdditionalProperties) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateLink(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateRecipeID(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *SolutionResponseAdditionalProperties) validateLink(formats strfmt.Registry) error {

	if err := validate.Required("link", "body", m.Link); err != nil {
		return err
	}

	if err := validate.FormatOf("link", "body", "uri", m.Link.String(), formats); err != nil {
		return err
	}

	return nil
}

func (m *SolutionResponseAdditionalProperties) validateRecipeID(formats strfmt.Registry) error {

	if err := validate.Required("recipe_id", "body", m.RecipeID); err != nil {
		return err
	}

	if err := validate.FormatOf("recipe_id", "body", "uuid", m.RecipeID.String(), formats); err != nil {
		return err
	}

	return nil
}

// ContextValidate validates this solution response additional properties based on context it is used
func (m *SolutionResponseAdditionalProperties) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *SolutionResponseAdditionalProperties) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *SolutionResponseAdditionalProperties) UnmarshalBinary(b []byte) error {
	var res SolutionResponseAdditionalProperties
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
