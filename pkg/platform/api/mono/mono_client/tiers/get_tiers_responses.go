// Code generated by go-swagger; DO NOT EDIT.

package tiers

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
)

// GetTiersReader is a Reader for the GetTiers structure.
type GetTiersReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *GetTiersReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewGetTiersOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 403:
		result := NewGetTiersForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 404:
		result := NewGetTiersNotFound()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 500:
		result := NewGetTiersInternalServerError()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		return nil, runtime.NewAPIError("response status code does not match any response statuses defined for this endpoint in the swagger spec", response, response.Code())
	}
}

// NewGetTiersOK creates a GetTiersOK with default headers values
func NewGetTiersOK() *GetTiersOK {
	return &GetTiersOK{}
}

/* GetTiersOK describes a response with status code 200, with default header values.

Success
*/
type GetTiersOK struct {
	Payload []*mono_models.Tier
}

func (o *GetTiersOK) Error() string {
	return fmt.Sprintf("[GET /tiers][%d] getTiersOK  %+v", 200, o.Payload)
}
func (o *GetTiersOK) GetPayload() []*mono_models.Tier {
	return o.Payload
}

func (o *GetTiersOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetTiersForbidden creates a GetTiersForbidden with default headers values
func NewGetTiersForbidden() *GetTiersForbidden {
	return &GetTiersForbidden{}
}

/* GetTiersForbidden describes a response with status code 403, with default header values.

Forbidden
*/
type GetTiersForbidden struct {
	Payload *mono_models.Message
}

func (o *GetTiersForbidden) Error() string {
	return fmt.Sprintf("[GET /tiers][%d] getTiersForbidden  %+v", 403, o.Payload)
}
func (o *GetTiersForbidden) GetPayload() *mono_models.Message {
	return o.Payload
}

func (o *GetTiersForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(mono_models.Message)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetTiersNotFound creates a GetTiersNotFound with default headers values
func NewGetTiersNotFound() *GetTiersNotFound {
	return &GetTiersNotFound{}
}

/* GetTiersNotFound describes a response with status code 404, with default header values.

No tiers available
*/
type GetTiersNotFound struct {
	Payload *mono_models.Message
}

func (o *GetTiersNotFound) Error() string {
	return fmt.Sprintf("[GET /tiers][%d] getTiersNotFound  %+v", 404, o.Payload)
}
func (o *GetTiersNotFound) GetPayload() *mono_models.Message {
	return o.Payload
}

func (o *GetTiersNotFound) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(mono_models.Message)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetTiersInternalServerError creates a GetTiersInternalServerError with default headers values
func NewGetTiersInternalServerError() *GetTiersInternalServerError {
	return &GetTiersInternalServerError{}
}

/* GetTiersInternalServerError describes a response with status code 500, with default header values.

Server Error
*/
type GetTiersInternalServerError struct {
	Payload *mono_models.Message
}

func (o *GetTiersInternalServerError) Error() string {
	return fmt.Sprintf("[GET /tiers][%d] getTiersInternalServerError  %+v", 500, o.Payload)
}
func (o *GetTiersInternalServerError) GetPayload() *mono_models.Message {
	return o.Payload
}

func (o *GetTiersInternalServerError) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(mono_models.Message)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
