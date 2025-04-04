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

// IngredientVersionCreateAllOf0 ingredient version create all of0
//
// swagger:model ingredientVersionCreateAllOf0
type IngredientVersionCreateAllOf0 struct {

	// The author(s) of this ingredient version, referenced by their author ID.
	// Required: true
	// Min Items: 1
	Authors []strfmt.UUID `json:"authors"`
}

// Validate validates this ingredient version create all of0
func (m *IngredientVersionCreateAllOf0) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateAuthors(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *IngredientVersionCreateAllOf0) validateAuthors(formats strfmt.Registry) error {

	if err := validate.Required("authors", "body", m.Authors); err != nil {
		return err
	}

	iAuthorsSize := int64(len(m.Authors))

	if err := validate.MinItems("authors", "body", iAuthorsSize, 1); err != nil {
		return err
	}

	for i := 0; i < len(m.Authors); i++ {

		if err := validate.FormatOf("authors"+"."+strconv.Itoa(i), "body", "uuid", m.Authors[i].String(), formats); err != nil {
			return err
		}

	}

	return nil
}

// ContextValidate validates this ingredient version create all of0 based on context it is used
func (m *IngredientVersionCreateAllOf0) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *IngredientVersionCreateAllOf0) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *IngredientVersionCreateAllOf0) UnmarshalBinary(b []byte) error {
	var res IngredientVersionCreateAllOf0
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
