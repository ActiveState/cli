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

// GetIngredientReader is a Reader for the GetIngredient structure.
type GetIngredientReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *GetIngredientReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewGetIngredientOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewGetIngredientDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewGetIngredientOK creates a GetIngredientOK with default headers values
func NewGetIngredientOK() *GetIngredientOK {
	return &GetIngredientOK{}
}

/* GetIngredientOK describes a response with status code 200, with default header values.

The retrieved ingredient
*/
type GetIngredientOK struct {
	Payload *inventory_models.Ingredient
}

func (o *GetIngredientOK) Error() string {
	return fmt.Sprintf("[GET /v1/ingredients/{ingredient_id}][%d] getIngredientOK  %+v", 200, o.Payload)
}
func (o *GetIngredientOK) GetPayload() *inventory_models.Ingredient {
	return o.Payload
}

func (o *GetIngredientOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(inventory_models.Ingredient)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetIngredientDefault creates a GetIngredientDefault with default headers values
func NewGetIngredientDefault(code int) *GetIngredientDefault {
	return &GetIngredientDefault{
		_statusCode: code,
	}
}

/* GetIngredientDefault describes a response with status code -1, with default header values.

generic error response
*/
type GetIngredientDefault struct {
	_statusCode int

	Payload *inventory_models.RestAPIError
}

// Code gets the status code for the get ingredient default response
func (o *GetIngredientDefault) Code() int {
	return o._statusCode
}

func (o *GetIngredientDefault) Error() string {
	return fmt.Sprintf("[GET /v1/ingredients/{ingredient_id}][%d] getIngredient default  %+v", o._statusCode, o.Payload)
}
func (o *GetIngredientDefault) GetPayload() *inventory_models.RestAPIError {
	return o.Payload
}

func (o *GetIngredientDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(inventory_models.RestAPIError)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
