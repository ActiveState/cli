// Code generated by go-swagger; DO NOT EDIT.

package inventory_models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// IngredientCore Ingredient Core
//
// A unique ingredient that can be used in a recipe. These properties are shared by all ingredient models.
//
// swagger:model ingredientCore
type IngredientCore struct {
	IngredientCoreAllOf0

	IngredientUpdate
}

// UnmarshalJSON unmarshals this object from a JSON structure
func (m *IngredientCore) UnmarshalJSON(raw []byte) error {
	// AO0
	var aO0 IngredientCoreAllOf0
	if err := swag.ReadJSON(raw, &aO0); err != nil {
		return err
	}
	m.IngredientCoreAllOf0 = aO0

	// AO1
	var aO1 IngredientUpdate
	if err := swag.ReadJSON(raw, &aO1); err != nil {
		return err
	}
	m.IngredientUpdate = aO1

	return nil
}

// MarshalJSON marshals this object to a JSON structure
func (m IngredientCore) MarshalJSON() ([]byte, error) {
	_parts := make([][]byte, 0, 2)

	aO0, err := swag.WriteJSON(m.IngredientCoreAllOf0)
	if err != nil {
		return nil, err
	}
	_parts = append(_parts, aO0)

	aO1, err := swag.WriteJSON(m.IngredientUpdate)
	if err != nil {
		return nil, err
	}
	_parts = append(_parts, aO1)
	return swag.ConcatJSON(_parts...), nil
}

// Validate validates this ingredient core
func (m *IngredientCore) Validate(formats strfmt.Registry) error {
	var res []error

	// validation for a type composition with IngredientCoreAllOf0
	if err := m.IngredientCoreAllOf0.Validate(formats); err != nil {
		res = append(res, err)
	}
	// validation for a type composition with IngredientUpdate
	if err := m.IngredientUpdate.Validate(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

// ContextValidate validate this ingredient core based on the context it is used
func (m *IngredientCore) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	// validation for a type composition with IngredientCoreAllOf0
	if err := m.IngredientCoreAllOf0.ContextValidate(ctx, formats); err != nil {
		res = append(res, err)
	}
	// validation for a type composition with IngredientUpdate
	if err := m.IngredientUpdate.ContextValidate(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

// MarshalBinary interface implementation
func (m *IngredientCore) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *IngredientCore) UnmarshalBinary(b []byte) error {
	var res IngredientCore
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
