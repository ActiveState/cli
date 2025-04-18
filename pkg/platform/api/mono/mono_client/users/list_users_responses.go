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

// ListUsersReader is a Reader for the ListUsers structure.
type ListUsersReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListUsersReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListUsersOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 500:
		result := NewListUsersInternalServerError()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		return nil, runtime.NewAPIError("response status code does not match any response statuses defined for this endpoint in the swagger spec", response, response.Code())
	}
}

// NewListUsersOK creates a ListUsersOK with default headers values
func NewListUsersOK() *ListUsersOK {
	return &ListUsersOK{}
}

/* ListUsersOK describes a response with status code 200, with default header values.

Success
*/
type ListUsersOK struct {
	Payload []*mono_models.User
}

func (o *ListUsersOK) Error() string {
	return fmt.Sprintf("[GET /users][%d] listUsersOK  %+v", 200, o.Payload)
}
func (o *ListUsersOK) GetPayload() []*mono_models.User {
	return o.Payload
}

func (o *ListUsersOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListUsersInternalServerError creates a ListUsersInternalServerError with default headers values
func NewListUsersInternalServerError() *ListUsersInternalServerError {
	return &ListUsersInternalServerError{}
}

/* ListUsersInternalServerError describes a response with status code 500, with default header values.

Server Error
*/
type ListUsersInternalServerError struct {
	Payload *mono_models.Message
}

func (o *ListUsersInternalServerError) Error() string {
	return fmt.Sprintf("[GET /users][%d] listUsersInternalServerError  %+v", 500, o.Payload)
}
func (o *ListUsersInternalServerError) GetPayload() *mono_models.Message {
	return o.Payload
}

func (o *ListUsersInternalServerError) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(mono_models.Message)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
