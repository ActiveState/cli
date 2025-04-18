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

// CPUArchitecturePagedList CPU Architecture Paged List
//
// A paginated list of CPU architectures
//
// swagger:model cpuArchitecturePagedList
type CPUArchitecturePagedList struct {

	// A page of CPU architectures
	// Required: true
	CPUArchitectures []*CPUArchitecture `json:"cpu_architectures"`

	// links
	// Required: true
	Links *PagingLinks `json:"links"`

	// paging
	// Required: true
	Paging *Paging `json:"paging"`
}

// Validate validates this cpu architecture paged list
func (m *CPUArchitecturePagedList) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateCPUArchitectures(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateLinks(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validatePaging(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *CPUArchitecturePagedList) validateCPUArchitectures(formats strfmt.Registry) error {

	if err := validate.Required("cpu_architectures", "body", m.CPUArchitectures); err != nil {
		return err
	}

	for i := 0; i < len(m.CPUArchitectures); i++ {
		if swag.IsZero(m.CPUArchitectures[i]) { // not required
			continue
		}

		if m.CPUArchitectures[i] != nil {
			if err := m.CPUArchitectures[i].Validate(formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("cpu_architectures" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

func (m *CPUArchitecturePagedList) validateLinks(formats strfmt.Registry) error {

	if err := validate.Required("links", "body", m.Links); err != nil {
		return err
	}

	if m.Links != nil {
		if err := m.Links.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("links")
			}
			return err
		}
	}

	return nil
}

func (m *CPUArchitecturePagedList) validatePaging(formats strfmt.Registry) error {

	if err := validate.Required("paging", "body", m.Paging); err != nil {
		return err
	}

	if m.Paging != nil {
		if err := m.Paging.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("paging")
			}
			return err
		}
	}

	return nil
}

// ContextValidate validate this cpu architecture paged list based on the context it is used
func (m *CPUArchitecturePagedList) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateCPUArchitectures(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateLinks(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidatePaging(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *CPUArchitecturePagedList) contextValidateCPUArchitectures(ctx context.Context, formats strfmt.Registry) error {

	for i := 0; i < len(m.CPUArchitectures); i++ {

		if m.CPUArchitectures[i] != nil {
			if err := m.CPUArchitectures[i].ContextValidate(ctx, formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("cpu_architectures" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

func (m *CPUArchitecturePagedList) contextValidateLinks(ctx context.Context, formats strfmt.Registry) error {

	if m.Links != nil {
		if err := m.Links.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("links")
			}
			return err
		}
	}

	return nil
}

func (m *CPUArchitecturePagedList) contextValidatePaging(ctx context.Context, formats strfmt.Registry) error {

	if m.Paging != nil {
		if err := m.Paging.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("paging")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (m *CPUArchitecturePagedList) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *CPUArchitecturePagedList) UnmarshalBinary(b []byte) error {
	var res CPUArchitecturePagedList
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
