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

// BulkInviteOrganizationReader is a Reader for the BulkInviteOrganization structure.
type BulkInviteOrganizationReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *BulkInviteOrganizationReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewBulkInviteOrganizationOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 400:
		result := NewBulkInviteOrganizationBadRequest()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 401:
		result := NewBulkInviteOrganizationUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewBulkInviteOrganizationForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 500:
		result := NewBulkInviteOrganizationInternalServerError()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		return nil, runtime.NewAPIError("response status code does not match any response statuses defined for this endpoint in the swagger spec", response, response.Code())
	}
}

// NewBulkInviteOrganizationOK creates a BulkInviteOrganizationOK with default headers values
func NewBulkInviteOrganizationOK() *BulkInviteOrganizationOK {
	return &BulkInviteOrganizationOK{}
}

/* BulkInviteOrganizationOK describes a response with status code 200, with default header values.

Success
*/
type BulkInviteOrganizationOK struct {
	Payload []*mono_models.BulkInvitationResponse
}

func (o *BulkInviteOrganizationOK) Error() string {
	return fmt.Sprintf("[POST /organizations/{organizationName}/invitations/bulk][%d] bulkInviteOrganizationOK  %+v", 200, o.Payload)
}
func (o *BulkInviteOrganizationOK) GetPayload() []*mono_models.BulkInvitationResponse {
	return o.Payload
}

func (o *BulkInviteOrganizationOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewBulkInviteOrganizationBadRequest creates a BulkInviteOrganizationBadRequest with default headers values
func NewBulkInviteOrganizationBadRequest() *BulkInviteOrganizationBadRequest {
	return &BulkInviteOrganizationBadRequest{}
}

/* BulkInviteOrganizationBadRequest describes a response with status code 400, with default header values.

Bad Request
*/
type BulkInviteOrganizationBadRequest struct {
	Payload *mono_models.Message
}

func (o *BulkInviteOrganizationBadRequest) Error() string {
	return fmt.Sprintf("[POST /organizations/{organizationName}/invitations/bulk][%d] bulkInviteOrganizationBadRequest  %+v", 400, o.Payload)
}
func (o *BulkInviteOrganizationBadRequest) GetPayload() *mono_models.Message {
	return o.Payload
}

func (o *BulkInviteOrganizationBadRequest) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(mono_models.Message)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewBulkInviteOrganizationUnauthorized creates a BulkInviteOrganizationUnauthorized with default headers values
func NewBulkInviteOrganizationUnauthorized() *BulkInviteOrganizationUnauthorized {
	return &BulkInviteOrganizationUnauthorized{}
}

/* BulkInviteOrganizationUnauthorized describes a response with status code 401, with default header values.

Unauthorized
*/
type BulkInviteOrganizationUnauthorized struct {
}

func (o *BulkInviteOrganizationUnauthorized) Error() string {
	return fmt.Sprintf("[POST /organizations/{organizationName}/invitations/bulk][%d] bulkInviteOrganizationUnauthorized ", 401)
}

func (o *BulkInviteOrganizationUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewBulkInviteOrganizationForbidden creates a BulkInviteOrganizationForbidden with default headers values
func NewBulkInviteOrganizationForbidden() *BulkInviteOrganizationForbidden {
	return &BulkInviteOrganizationForbidden{}
}

/* BulkInviteOrganizationForbidden describes a response with status code 403, with default header values.

Forbidden
*/
type BulkInviteOrganizationForbidden struct {
	Payload *mono_models.Message
}

func (o *BulkInviteOrganizationForbidden) Error() string {
	return fmt.Sprintf("[POST /organizations/{organizationName}/invitations/bulk][%d] bulkInviteOrganizationForbidden  %+v", 403, o.Payload)
}
func (o *BulkInviteOrganizationForbidden) GetPayload() *mono_models.Message {
	return o.Payload
}

func (o *BulkInviteOrganizationForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(mono_models.Message)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewBulkInviteOrganizationInternalServerError creates a BulkInviteOrganizationInternalServerError with default headers values
func NewBulkInviteOrganizationInternalServerError() *BulkInviteOrganizationInternalServerError {
	return &BulkInviteOrganizationInternalServerError{}
}

/* BulkInviteOrganizationInternalServerError describes a response with status code 500, with default header values.

Server Error
*/
type BulkInviteOrganizationInternalServerError struct {
	Payload *mono_models.Message
}

func (o *BulkInviteOrganizationInternalServerError) Error() string {
	return fmt.Sprintf("[POST /organizations/{organizationName}/invitations/bulk][%d] bulkInviteOrganizationInternalServerError  %+v", 500, o.Payload)
}
func (o *BulkInviteOrganizationInternalServerError) GetPayload() *mono_models.Message {
	return o.Payload
}

func (o *BulkInviteOrganizationInternalServerError) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(mono_models.Message)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
