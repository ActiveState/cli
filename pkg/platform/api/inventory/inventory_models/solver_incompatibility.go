// Code generated by go-swagger; DO NOT EDIT.

package inventory_models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"encoding/json"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// SolverIncompatibility Solver Incompatibility
//
// A requirement, transitive dependency or platform that is part of a solver error
//
// swagger:model solverIncompatibility
type SolverIncompatibility struct {

	// The name of the dependency or requirement feature
	Feature string `json:"feature,omitempty"`

	// The name of the dependency or requirement namespace
	Namespace string `json:"namespace,omitempty"`

	// The id of the platform
	// Format: uuid
	PlatformID strfmt.UUID `json:"platform_id,omitempty"`

	// The name of the platform's kernel
	PlatformKernel string `json:"platform_kernel,omitempty"`

	// The type of this incompatibility
	// Required: true
	// Enum: [dependency platform requirement]
	Type *string `json:"type"`
}

// Validate validates this solver incompatibility
func (m *SolverIncompatibility) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validatePlatformID(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateType(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *SolverIncompatibility) validatePlatformID(formats strfmt.Registry) error {
	if swag.IsZero(m.PlatformID) { // not required
		return nil
	}

	if err := validate.FormatOf("platform_id", "body", "uuid", m.PlatformID.String(), formats); err != nil {
		return err
	}

	return nil
}

var solverIncompatibilityTypeTypePropEnum []interface{}

func init() {
	var res []string
	if err := json.Unmarshal([]byte(`["dependency","platform","requirement"]`), &res); err != nil {
		panic(err)
	}
	for _, v := range res {
		solverIncompatibilityTypeTypePropEnum = append(solverIncompatibilityTypeTypePropEnum, v)
	}
}

const (

	// SolverIncompatibilityTypeDependency captures enum value "dependency"
	SolverIncompatibilityTypeDependency string = "dependency"

	// SolverIncompatibilityTypePlatform captures enum value "platform"
	SolverIncompatibilityTypePlatform string = "platform"

	// SolverIncompatibilityTypeRequirement captures enum value "requirement"
	SolverIncompatibilityTypeRequirement string = "requirement"
)

// prop value enum
func (m *SolverIncompatibility) validateTypeEnum(path, location string, value string) error {
	if err := validate.EnumCase(path, location, value, solverIncompatibilityTypeTypePropEnum, true); err != nil {
		return err
	}
	return nil
}

func (m *SolverIncompatibility) validateType(formats strfmt.Registry) error {

	if err := validate.Required("type", "body", m.Type); err != nil {
		return err
	}

	// value enum
	if err := m.validateTypeEnum("type", "body", *m.Type); err != nil {
		return err
	}

	return nil
}

// ContextValidate validates this solver incompatibility based on context it is used
func (m *SolverIncompatibility) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *SolverIncompatibility) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *SolverIncompatibility) UnmarshalBinary(b []byte) error {
	var res SolverIncompatibility
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
