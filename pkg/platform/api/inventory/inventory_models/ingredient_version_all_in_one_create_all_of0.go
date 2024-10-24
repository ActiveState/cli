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

// IngredientVersionAllInOneCreateAllOf0 ingredient version all in one create all of0
//
// swagger:model ingredientVersionAllInOneCreateAllOf0
type IngredientVersionAllInOneCreateAllOf0 struct {

	// authors
	Authors []strfmt.UUID `json:"authors"`
}

// Validate validates this ingredient version all in one create all of0
func (m *IngredientVersionAllInOneCreateAllOf0) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateAuthors(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *IngredientVersionAllInOneCreateAllOf0) validateAuthors(formats strfmt.Registry) error {
	if swag.IsZero(m.Authors) { // not required
		return nil
	}

	for i := 0; i < len(m.Authors); i++ {

		if err := validate.FormatOf("authors"+"."+strconv.Itoa(i), "body", "uuid", m.Authors[i].String(), formats); err != nil {
			return err
		}

	}

	return nil
}

// ContextValidate validates this ingredient version all in one create all of0 based on context it is used
func (m *IngredientVersionAllInOneCreateAllOf0) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *IngredientVersionAllInOneCreateAllOf0) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *IngredientVersionAllInOneCreateAllOf0) UnmarshalBinary(b []byte) error {
	var res IngredientVersionAllInOneCreateAllOf0
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
