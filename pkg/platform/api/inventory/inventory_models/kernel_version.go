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

// KernelVersion Kernel Version
//
// The full kernel version data model
//
// swagger:model kernelVersion
type KernelVersion struct {
	KernelVersionAllOf0

	KernelVersionCore

	RevisionedResource
}

// UnmarshalJSON unmarshals this object from a JSON structure
func (m *KernelVersion) UnmarshalJSON(raw []byte) error {
	// AO0
	var aO0 KernelVersionAllOf0
	if err := swag.ReadJSON(raw, &aO0); err != nil {
		return err
	}
	m.KernelVersionAllOf0 = aO0

	// AO1
	var aO1 KernelVersionCore
	if err := swag.ReadJSON(raw, &aO1); err != nil {
		return err
	}
	m.KernelVersionCore = aO1

	// AO2
	var aO2 RevisionedResource
	if err := swag.ReadJSON(raw, &aO2); err != nil {
		return err
	}
	m.RevisionedResource = aO2

	return nil
}

// MarshalJSON marshals this object to a JSON structure
func (m KernelVersion) MarshalJSON() ([]byte, error) {
	_parts := make([][]byte, 0, 3)

	aO0, err := swag.WriteJSON(m.KernelVersionAllOf0)
	if err != nil {
		return nil, err
	}
	_parts = append(_parts, aO0)

	aO1, err := swag.WriteJSON(m.KernelVersionCore)
	if err != nil {
		return nil, err
	}
	_parts = append(_parts, aO1)

	aO2, err := swag.WriteJSON(m.RevisionedResource)
	if err != nil {
		return nil, err
	}
	_parts = append(_parts, aO2)
	return swag.ConcatJSON(_parts...), nil
}

// Validate validates this kernel version
func (m *KernelVersion) Validate(formats strfmt.Registry) error {
	var res []error

	// validation for a type composition with KernelVersionAllOf0
	if err := m.KernelVersionAllOf0.Validate(formats); err != nil {
		res = append(res, err)
	}
	// validation for a type composition with KernelVersionCore
	if err := m.KernelVersionCore.Validate(formats); err != nil {
		res = append(res, err)
	}
	// validation for a type composition with RevisionedResource
	if err := m.RevisionedResource.Validate(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

// ContextValidate validate this kernel version based on the context it is used
func (m *KernelVersion) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	// validation for a type composition with KernelVersionAllOf0
	if err := m.KernelVersionAllOf0.ContextValidate(ctx, formats); err != nil {
		res = append(res, err)
	}
	// validation for a type composition with KernelVersionCore
	if err := m.KernelVersionCore.ContextValidate(ctx, formats); err != nil {
		res = append(res, err)
	}
	// validation for a type composition with RevisionedResource
	if err := m.RevisionedResource.ContextValidate(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

// MarshalBinary interface implementation
func (m *KernelVersion) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *KernelVersion) UnmarshalBinary(b []byte) error {
	var res KernelVersion
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
