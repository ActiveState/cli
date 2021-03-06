// Code generated by go-swagger; DO NOT EDIT.

package authentication

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
)

// ResetPasswordReader is a Reader for the ResetPassword structure.
type ResetPasswordReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ResetPasswordReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewResetPasswordOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 400:
		result := NewResetPasswordBadRequest()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewResetPasswordForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 500:
		result := NewResetPasswordInternalServerError()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		return nil, runtime.NewAPIError("response status code does not match any response statuses defined for this endpoint in the swagger spec", response, response.Code())
	}
}

// NewResetPasswordOK creates a ResetPasswordOK with default headers values
func NewResetPasswordOK() *ResetPasswordOK {
	return &ResetPasswordOK{}
}

/* ResetPasswordOK describes a response with status code 200, with default header values.

Success
*/
type ResetPasswordOK struct {
	Payload *mono_models.Message
}

func (o *ResetPasswordOK) Error() string {
	return fmt.Sprintf("[POST /reset-password][%d] resetPasswordOK  %+v", 200, o.Payload)
}
func (o *ResetPasswordOK) GetPayload() *mono_models.Message {
	return o.Payload
}

func (o *ResetPasswordOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(mono_models.Message)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewResetPasswordBadRequest creates a ResetPasswordBadRequest with default headers values
func NewResetPasswordBadRequest() *ResetPasswordBadRequest {
	return &ResetPasswordBadRequest{}
}

/* ResetPasswordBadRequest describes a response with status code 400, with default header values.

Bad Request
*/
type ResetPasswordBadRequest struct {
	Payload *mono_models.Message
}

func (o *ResetPasswordBadRequest) Error() string {
	return fmt.Sprintf("[POST /reset-password][%d] resetPasswordBadRequest  %+v", 400, o.Payload)
}
func (o *ResetPasswordBadRequest) GetPayload() *mono_models.Message {
	return o.Payload
}

func (o *ResetPasswordBadRequest) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(mono_models.Message)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewResetPasswordForbidden creates a ResetPasswordForbidden with default headers values
func NewResetPasswordForbidden() *ResetPasswordForbidden {
	return &ResetPasswordForbidden{}
}

/* ResetPasswordForbidden describes a response with status code 403, with default header values.

Forbidden
*/
type ResetPasswordForbidden struct {
	Payload *mono_models.Message
}

func (o *ResetPasswordForbidden) Error() string {
	return fmt.Sprintf("[POST /reset-password][%d] resetPasswordForbidden  %+v", 403, o.Payload)
}
func (o *ResetPasswordForbidden) GetPayload() *mono_models.Message {
	return o.Payload
}

func (o *ResetPasswordForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(mono_models.Message)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewResetPasswordInternalServerError creates a ResetPasswordInternalServerError with default headers values
func NewResetPasswordInternalServerError() *ResetPasswordInternalServerError {
	return &ResetPasswordInternalServerError{}
}

/* ResetPasswordInternalServerError describes a response with status code 500, with default header values.

Server Error
*/
type ResetPasswordInternalServerError struct {
	Payload *mono_models.Message
}

func (o *ResetPasswordInternalServerError) Error() string {
	return fmt.Sprintf("[POST /reset-password][%d] resetPasswordInternalServerError  %+v", 500, o.Payload)
}
func (o *ResetPasswordInternalServerError) GetPayload() *mono_models.Message {
	return o.Payload
}

func (o *ResetPasswordInternalServerError) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(mono_models.Message)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
