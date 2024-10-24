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

// DeviceCode device code
//
// swagger:model DeviceCode
type DeviceCode struct {

	// Code used to identify client when client polls for credentials
	// Required: true
	DeviceCode *string `json:"device_code"`

	// The lifetime in seconds of the "device_code" and "user_code".
	// Required: true
	ExpiresIn *string `json:"expires_in"`

	// Interval at which to poll for credentials
	Interval int64 `json:"interval,omitempty"`

	// If true tells client not to use polling.
	Nopoll bool `json:"nopoll,omitempty"`

	// Code to be enter when user opens verification_uri
	// Required: true
	UserCode *string `json:"user_code"`

	// The URI that gets opened in the browser by user or cli tool which includes the user_code as a query parameter in the URI.  This page MUST request the user to acknowledge they are authorizing a CLI tool to use their credentials to query the Platform API.  So a BUTTON that says “Authorize”
	// Required: true
	VerificationURIComplete *string `json:"verification_uri_complete"`
}

// Validate validates this device code
func (m *DeviceCode) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateDeviceCode(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateExpiresIn(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateUserCode(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateVerificationURIComplete(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *DeviceCode) validateDeviceCode(formats strfmt.Registry) error {

	if err := validate.Required("device_code", "body", m.DeviceCode); err != nil {
		return err
	}

	return nil
}

func (m *DeviceCode) validateExpiresIn(formats strfmt.Registry) error {

	if err := validate.Required("expires_in", "body", m.ExpiresIn); err != nil {
		return err
	}

	return nil
}

func (m *DeviceCode) validateUserCode(formats strfmt.Registry) error {

	if err := validate.Required("user_code", "body", m.UserCode); err != nil {
		return err
	}

	return nil
}

func (m *DeviceCode) validateVerificationURIComplete(formats strfmt.Registry) error {

	if err := validate.Required("verification_uri_complete", "body", m.VerificationURIComplete); err != nil {
		return err
	}

	return nil
}

// ContextValidate validates this device code based on context it is used
func (m *DeviceCode) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *DeviceCode) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *DeviceCode) UnmarshalBinary(b []byte) error {
	var res DeviceCode
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
