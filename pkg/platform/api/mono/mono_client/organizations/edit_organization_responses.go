// Code generated by go-swagger; DO NOT EDIT.

package organizations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
)

// EditOrganizationReader is a Reader for the EditOrganization structure.
type EditOrganizationReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *EditOrganizationReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewEditOrganizationOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 400:
		result := NewEditOrganizationBadRequest()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewEditOrganizationForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 404:
		result := NewEditOrganizationNotFound()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 500:
		result := NewEditOrganizationInternalServerError()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		return nil, runtime.NewAPIError("response status code does not match any response statuses defined for this endpoint in the swagger spec", response, response.Code())
	}
}

// NewEditOrganizationOK creates a EditOrganizationOK with default headers values
func NewEditOrganizationOK() *EditOrganizationOK {
	return &EditOrganizationOK{}
}

/* EditOrganizationOK describes a response with status code 200, with default header values.

Organization updated
*/
type EditOrganizationOK struct {
	Payload *mono_models.Organization
}

func (o *EditOrganizationOK) Error() string {
	return fmt.Sprintf("[POST /organizations/{organizationIdentifier}][%d] editOrganizationOK  %+v", 200, o.Payload)
}
func (o *EditOrganizationOK) GetPayload() *mono_models.Organization {
	return o.Payload
}

func (o *EditOrganizationOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(mono_models.Organization)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewEditOrganizationBadRequest creates a EditOrganizationBadRequest with default headers values
func NewEditOrganizationBadRequest() *EditOrganizationBadRequest {
	return &EditOrganizationBadRequest{}
}

/* EditOrganizationBadRequest describes a response with status code 400, with default header values.

Bad Request
*/
type EditOrganizationBadRequest struct {
	Payload *mono_models.Message
}

func (o *EditOrganizationBadRequest) Error() string {
	return fmt.Sprintf("[POST /organizations/{organizationIdentifier}][%d] editOrganizationBadRequest  %+v", 400, o.Payload)
}
func (o *EditOrganizationBadRequest) GetPayload() *mono_models.Message {
	return o.Payload
}

func (o *EditOrganizationBadRequest) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(mono_models.Message)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewEditOrganizationForbidden creates a EditOrganizationForbidden with default headers values
func NewEditOrganizationForbidden() *EditOrganizationForbidden {
	return &EditOrganizationForbidden{}
}

/* EditOrganizationForbidden describes a response with status code 403, with default header values.

Unauthorized
*/
type EditOrganizationForbidden struct {
	Payload *mono_models.Message
}

func (o *EditOrganizationForbidden) Error() string {
	return fmt.Sprintf("[POST /organizations/{organizationIdentifier}][%d] editOrganizationForbidden  %+v", 403, o.Payload)
}
func (o *EditOrganizationForbidden) GetPayload() *mono_models.Message {
	return o.Payload
}

func (o *EditOrganizationForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(mono_models.Message)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewEditOrganizationNotFound creates a EditOrganizationNotFound with default headers values
func NewEditOrganizationNotFound() *EditOrganizationNotFound {
	return &EditOrganizationNotFound{}
}

/* EditOrganizationNotFound describes a response with status code 404, with default header values.

Not Found
*/
type EditOrganizationNotFound struct {
	Payload *mono_models.Message
}

func (o *EditOrganizationNotFound) Error() string {
	return fmt.Sprintf("[POST /organizations/{organizationIdentifier}][%d] editOrganizationNotFound  %+v", 404, o.Payload)
}
func (o *EditOrganizationNotFound) GetPayload() *mono_models.Message {
	return o.Payload
}

func (o *EditOrganizationNotFound) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(mono_models.Message)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewEditOrganizationInternalServerError creates a EditOrganizationInternalServerError with default headers values
func NewEditOrganizationInternalServerError() *EditOrganizationInternalServerError {
	return &EditOrganizationInternalServerError{}
}

/* EditOrganizationInternalServerError describes a response with status code 500, with default header values.

Server Error
*/
type EditOrganizationInternalServerError struct {
	Payload *mono_models.Message
}

func (o *EditOrganizationInternalServerError) Error() string {
	return fmt.Sprintf("[POST /organizations/{organizationIdentifier}][%d] editOrganizationInternalServerError  %+v", 500, o.Payload)
}
func (o *EditOrganizationInternalServerError) GetPayload() *mono_models.Message {
	return o.Payload
}

func (o *EditOrganizationInternalServerError) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(mono_models.Message)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
