// Code generated by go-swagger; DO NOT EDIT.

package users

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
)

// GetEmailVerificationLinkReader is a Reader for the GetEmailVerificationLink structure.
type GetEmailVerificationLinkReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *GetEmailVerificationLinkReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewGetEmailVerificationLinkOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 400:
		result := NewGetEmailVerificationLinkBadRequest()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewGetEmailVerificationLinkForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 404:
		result := NewGetEmailVerificationLinkNotFound()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 500:
		result := NewGetEmailVerificationLinkInternalServerError()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		return nil, runtime.NewAPIError("response status code does not match any response statuses defined for this endpoint in the swagger spec", response, response.Code())
	}
}

// NewGetEmailVerificationLinkOK creates a GetEmailVerificationLinkOK with default headers values
func NewGetEmailVerificationLinkOK() *GetEmailVerificationLinkOK {
	return &GetEmailVerificationLinkOK{}
}

/* GetEmailVerificationLinkOK describes a response with status code 200, with default header values.

Success
*/
type GetEmailVerificationLinkOK struct {
	Payload string
}

func (o *GetEmailVerificationLinkOK) Error() string {
	return fmt.Sprintf("[GET /users/verification/{email}][%d] getEmailVerificationLinkOK  %+v", 200, o.Payload)
}
func (o *GetEmailVerificationLinkOK) GetPayload() string {
	return o.Payload
}

func (o *GetEmailVerificationLinkOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetEmailVerificationLinkBadRequest creates a GetEmailVerificationLinkBadRequest with default headers values
func NewGetEmailVerificationLinkBadRequest() *GetEmailVerificationLinkBadRequest {
	return &GetEmailVerificationLinkBadRequest{}
}

/* GetEmailVerificationLinkBadRequest describes a response with status code 400, with default header values.

Email is already verified
*/
type GetEmailVerificationLinkBadRequest struct {
	Payload *mono_models.Message
}

func (o *GetEmailVerificationLinkBadRequest) Error() string {
	return fmt.Sprintf("[GET /users/verification/{email}][%d] getEmailVerificationLinkBadRequest  %+v", 400, o.Payload)
}
func (o *GetEmailVerificationLinkBadRequest) GetPayload() *mono_models.Message {
	return o.Payload
}

func (o *GetEmailVerificationLinkBadRequest) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(mono_models.Message)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetEmailVerificationLinkForbidden creates a GetEmailVerificationLinkForbidden with default headers values
func NewGetEmailVerificationLinkForbidden() *GetEmailVerificationLinkForbidden {
	return &GetEmailVerificationLinkForbidden{}
}

/* GetEmailVerificationLinkForbidden describes a response with status code 403, with default header values.

Forbidden
*/
type GetEmailVerificationLinkForbidden struct {
	Payload *mono_models.Message
}

func (o *GetEmailVerificationLinkForbidden) Error() string {
	return fmt.Sprintf("[GET /users/verification/{email}][%d] getEmailVerificationLinkForbidden  %+v", 403, o.Payload)
}
func (o *GetEmailVerificationLinkForbidden) GetPayload() *mono_models.Message {
	return o.Payload
}

func (o *GetEmailVerificationLinkForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(mono_models.Message)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetEmailVerificationLinkNotFound creates a GetEmailVerificationLinkNotFound with default headers values
func NewGetEmailVerificationLinkNotFound() *GetEmailVerificationLinkNotFound {
	return &GetEmailVerificationLinkNotFound{}
}

/* GetEmailVerificationLinkNotFound describes a response with status code 404, with default header values.

Email not found
*/
type GetEmailVerificationLinkNotFound struct {
	Payload *mono_models.Message
}

func (o *GetEmailVerificationLinkNotFound) Error() string {
	return fmt.Sprintf("[GET /users/verification/{email}][%d] getEmailVerificationLinkNotFound  %+v", 404, o.Payload)
}
func (o *GetEmailVerificationLinkNotFound) GetPayload() *mono_models.Message {
	return o.Payload
}

func (o *GetEmailVerificationLinkNotFound) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(mono_models.Message)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetEmailVerificationLinkInternalServerError creates a GetEmailVerificationLinkInternalServerError with default headers values
func NewGetEmailVerificationLinkInternalServerError() *GetEmailVerificationLinkInternalServerError {
	return &GetEmailVerificationLinkInternalServerError{}
}

/* GetEmailVerificationLinkInternalServerError describes a response with status code 500, with default header values.

Server Error
*/
type GetEmailVerificationLinkInternalServerError struct {
	Payload *mono_models.Message
}

func (o *GetEmailVerificationLinkInternalServerError) Error() string {
	return fmt.Sprintf("[GET /users/verification/{email}][%d] getEmailVerificationLinkInternalServerError  %+v", 500, o.Payload)
}
func (o *GetEmailVerificationLinkInternalServerError) GetPayload() *mono_models.Message {
	return o.Payload
}

func (o *GetEmailVerificationLinkInternalServerError) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(mono_models.Message)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
