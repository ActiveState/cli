// Code generated by go-swagger; DO NOT EDIT.

package keys

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"

	secrets_models "github.com/ActiveState/cli/pkg/platform/api/secrets/secrets_models"
)

// SaveKeypairReader is a Reader for the SaveKeypair structure.
type SaveKeypairReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *SaveKeypairReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {

	case 204:
		result := NewSaveKeypairNoContent()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil

	case 400:
		result := NewSaveKeypairBadRequest()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result

	case 401:
		result := NewSaveKeypairUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result

	case 500:
		result := NewSaveKeypairInternalServerError()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result

	default:
		return nil, runtime.NewAPIError("unknown error", response, response.Code())
	}
}

// NewSaveKeypairNoContent creates a SaveKeypairNoContent with default headers values
func NewSaveKeypairNoContent() *SaveKeypairNoContent {
	return &SaveKeypairNoContent{}
}

/*SaveKeypairNoContent handles this case with default header values.

Success
*/
type SaveKeypairNoContent struct {
}

func (o *SaveKeypairNoContent) Error() string {
	return fmt.Sprintf("[PUT /keypair][%d] saveKeypairNoContent ", 204)
}

func (o *SaveKeypairNoContent) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewSaveKeypairBadRequest creates a SaveKeypairBadRequest with default headers values
func NewSaveKeypairBadRequest() *SaveKeypairBadRequest {
	return &SaveKeypairBadRequest{}
}

/*SaveKeypairBadRequest handles this case with default header values.

Bad Request
*/
type SaveKeypairBadRequest struct {
	Payload *secrets_models.Message
}

func (o *SaveKeypairBadRequest) Error() string {
	return fmt.Sprintf("[PUT /keypair][%d] saveKeypairBadRequest  %+v", 400, o.Payload)
}

func (o *SaveKeypairBadRequest) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(secrets_models.Message)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewSaveKeypairUnauthorized creates a SaveKeypairUnauthorized with default headers values
func NewSaveKeypairUnauthorized() *SaveKeypairUnauthorized {
	return &SaveKeypairUnauthorized{}
}

/*SaveKeypairUnauthorized handles this case with default header values.

Invalid credentials
*/
type SaveKeypairUnauthorized struct {
	Payload *secrets_models.Message
}

func (o *SaveKeypairUnauthorized) Error() string {
	return fmt.Sprintf("[PUT /keypair][%d] saveKeypairUnauthorized  %+v", 401, o.Payload)
}

func (o *SaveKeypairUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(secrets_models.Message)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewSaveKeypairInternalServerError creates a SaveKeypairInternalServerError with default headers values
func NewSaveKeypairInternalServerError() *SaveKeypairInternalServerError {
	return &SaveKeypairInternalServerError{}
}

/*SaveKeypairInternalServerError handles this case with default header values.

Server Error
*/
type SaveKeypairInternalServerError struct {
	Payload *secrets_models.Message
}

func (o *SaveKeypairInternalServerError) Error() string {
	return fmt.Sprintf("[PUT /keypair][%d] saveKeypairInternalServerError  %+v", 500, o.Payload)
}

func (o *SaveKeypairInternalServerError) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(secrets_models.Message)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
