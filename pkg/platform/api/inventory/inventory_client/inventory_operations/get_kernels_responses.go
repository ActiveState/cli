// Code generated by go-swagger; DO NOT EDIT.

package inventory_operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
)

// GetKernelsReader is a Reader for the GetKernels structure.
type GetKernelsReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *GetKernelsReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewGetKernelsOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewGetKernelsDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewGetKernelsOK creates a GetKernelsOK with default headers values
func NewGetKernelsOK() *GetKernelsOK {
	return &GetKernelsOK{}
}

/* GetKernelsOK describes a response with status code 200, with default header values.

A paginated list of kernels
*/
type GetKernelsOK struct {
	Payload *inventory_models.KernelPagedList
}

func (o *GetKernelsOK) Error() string {
	return fmt.Sprintf("[GET /v1/kernels][%d] getKernelsOK  %+v", 200, o.Payload)
}
func (o *GetKernelsOK) GetPayload() *inventory_models.KernelPagedList {
	return o.Payload
}

func (o *GetKernelsOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(inventory_models.KernelPagedList)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetKernelsDefault creates a GetKernelsDefault with default headers values
func NewGetKernelsDefault(code int) *GetKernelsDefault {
	return &GetKernelsDefault{
		_statusCode: code,
	}
}

/* GetKernelsDefault describes a response with status code -1, with default header values.

generic error response
*/
type GetKernelsDefault struct {
	_statusCode int

	Payload *inventory_models.RestAPIError
}

// Code gets the status code for the get kernels default response
func (o *GetKernelsDefault) Code() int {
	return o._statusCode
}

func (o *GetKernelsDefault) Error() string {
	return fmt.Sprintf("[GET /v1/kernels][%d] getKernels default  %+v", o._statusCode, o.Payload)
}
func (o *GetKernelsDefault) GetPayload() *inventory_models.RestAPIError {
	return o.Payload
}

func (o *GetKernelsDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(inventory_models.RestAPIError)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
