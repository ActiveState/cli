// Code generated by go-swagger; DO NOT EDIT.

package mono_models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// TagEditable tagEditable
//
// swagger:model TagEditable
type TagEditable struct {

	// The commit that this tag is currently pointing at
	// Format: uuid
	CommitID *strfmt.UUID `json:"commitID,omitempty"`

	// The human readable label or name of the tag.
	Label *string `json:"label,omitempty"`
}

// Validate validates this tag editable
func (m *TagEditable) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateCommitID(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *TagEditable) validateCommitID(formats strfmt.Registry) error {
	if swag.IsZero(m.CommitID) { // not required
		return nil
	}

	if err := validate.FormatOf("commitID", "body", "uuid", m.CommitID.String(), formats); err != nil {
		return err
	}

	return nil
}

// ContextValidate validates this tag editable based on context it is used
func (m *TagEditable) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *TagEditable) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *TagEditable) UnmarshalBinary(b []byte) error {
	var res TagEditable
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
