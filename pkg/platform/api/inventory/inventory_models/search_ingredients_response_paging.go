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

// SearchIngredientsResponsePaging search ingredients response paging
//
// swagger:model searchIngredientsResponsePaging
type SearchIngredientsResponsePaging struct {

	// The number of ingredients on this page
	// Minimum: 0
	ItemCount *int64 `json:"item_count,omitempty"`

	// The maximum number of ingredients that could be returned
	// Minimum: 1
	Limit int64 `json:"limit,omitempty"`

	// The number of ingredients skipped
	// Minimum: 0
	Offset *int64 `json:"offset,omitempty"`
}

// Validate validates this search ingredients response paging
func (m *SearchIngredientsResponsePaging) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateItemCount(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateLimit(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateOffset(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *SearchIngredientsResponsePaging) validateItemCount(formats strfmt.Registry) error {
	if swag.IsZero(m.ItemCount) { // not required
		return nil
	}

	if err := validate.MinimumInt("item_count", "body", *m.ItemCount, 0, false); err != nil {
		return err
	}

	return nil
}

func (m *SearchIngredientsResponsePaging) validateLimit(formats strfmt.Registry) error {
	if swag.IsZero(m.Limit) { // not required
		return nil
	}

	if err := validate.MinimumInt("limit", "body", m.Limit, 1, false); err != nil {
		return err
	}

	return nil
}

func (m *SearchIngredientsResponsePaging) validateOffset(formats strfmt.Registry) error {
	if swag.IsZero(m.Offset) { // not required
		return nil
	}

	if err := validate.MinimumInt("offset", "body", *m.Offset, 0, false); err != nil {
		return err
	}

	return nil
}

// ContextValidate validates this search ingredients response paging based on context it is used
func (m *SearchIngredientsResponsePaging) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *SearchIngredientsResponsePaging) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *SearchIngredientsResponsePaging) UnmarshalBinary(b []byte) error {
	var res SearchIngredientsResponsePaging
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
