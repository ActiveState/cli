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

// UniqueUsernameReader is a Reader for the UniqueUsername structure.
type UniqueUsernameReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *UniqueUsernameReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewUniqueUsernameOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 400:
		result := NewUniqueUsernameBadRequest()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 409:
		result := NewUniqueUsernameConflict()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		return nil, runtime.NewAPIError("response status code does not match any response statuses defined for this endpoint in the swagger spec", response, response.Code())
	}
}

// NewUniqueUsernameOK creates a UniqueUsernameOK with default headers values
func NewUniqueUsernameOK() *UniqueUsernameOK {
	return &UniqueUsernameOK{}
}

/* UniqueUsernameOK describes a response with status code 200, with default header values.

Username available
*/
type UniqueUsernameOK struct {
	Payload *mono_models.Message
}

func (o *UniqueUsernameOK) Error() string {
	return fmt.Sprintf("[GET /users/uniqueUsername/{username}][%d] uniqueUsernameOK  %+v", 200, o.Payload)
}
func (o *UniqueUsernameOK) GetPayload() *mono_models.Message {
	return o.Payload
}

func (o *UniqueUsernameOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(mono_models.Message)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewUniqueUsernameBadRequest creates a UniqueUsernameBadRequest with default headers values
func NewUniqueUsernameBadRequest() *UniqueUsernameBadRequest {
	return &UniqueUsernameBadRequest{}
}

/* UniqueUsernameBadRequest describes a response with status code 400, with default header values.

Bad Request
*/
type UniqueUsernameBadRequest struct {
	Payload *mono_models.Message
}

func (o *UniqueUsernameBadRequest) Error() string {
	return fmt.Sprintf("[GET /users/uniqueUsername/{username}][%d] uniqueUsernameBadRequest  %+v", 400, o.Payload)
}
func (o *UniqueUsernameBadRequest) GetPayload() *mono_models.Message {
	return o.Payload
}

func (o *UniqueUsernameBadRequest) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(mono_models.Message)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewUniqueUsernameConflict creates a UniqueUsernameConflict with default headers values
func NewUniqueUsernameConflict() *UniqueUsernameConflict {
	return &UniqueUsernameConflict{}
}

/* UniqueUsernameConflict describes a response with status code 409, with default header values.

Username Conflict
*/
type UniqueUsernameConflict struct {
	Payload *mono_models.Message
}

func (o *UniqueUsernameConflict) Error() string {
	return fmt.Sprintf("[GET /users/uniqueUsername/{username}][%d] uniqueUsernameConflict  %+v", 409, o.Payload)
}
func (o *UniqueUsernameConflict) GetPayload() *mono_models.Message {
	return o.Payload
}

func (o *UniqueUsernameConflict) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(mono_models.Message)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
