// Code generated by go-swagger; DO NOT EDIT.

package mono_models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"strconv"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// GithubLogin github login
//
// swagger:model GithubLogin
type GithubLogin struct {

	// name
	Name string `json:"name,omitempty"`

	// repositories
	Repositories []*GithubRepo `json:"repositories"`
}

// Validate validates this github login
func (m *GithubLogin) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateRepositories(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *GithubLogin) validateRepositories(formats strfmt.Registry) error {
	if swag.IsZero(m.Repositories) { // not required
		return nil
	}

	for i := 0; i < len(m.Repositories); i++ {
		if swag.IsZero(m.Repositories[i]) { // not required
			continue
		}

		if m.Repositories[i] != nil {
			if err := m.Repositories[i].Validate(formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("repositories" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

// ContextValidate validate this github login based on the context it is used
func (m *GithubLogin) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateRepositories(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *GithubLogin) contextValidateRepositories(ctx context.Context, formats strfmt.Registry) error {

	for i := 0; i < len(m.Repositories); i++ {

		if m.Repositories[i] != nil {
			if err := m.Repositories[i].ContextValidate(ctx, formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("repositories" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

// MarshalBinary interface implementation
func (m *GithubLogin) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *GithubLogin) UnmarshalBinary(b []byte) error {
	var res GithubLogin
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
