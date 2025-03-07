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

// GetIngredientVersionIngredientOptionSetsReader is a Reader for the GetIngredientVersionIngredientOptionSets structure.
type GetIngredientVersionIngredientOptionSetsReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *GetIngredientVersionIngredientOptionSetsReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewGetIngredientVersionIngredientOptionSetsOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewGetIngredientVersionIngredientOptionSetsDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewGetIngredientVersionIngredientOptionSetsOK creates a GetIngredientVersionIngredientOptionSetsOK with default headers values
func NewGetIngredientVersionIngredientOptionSetsOK() *GetIngredientVersionIngredientOptionSetsOK {
	return &GetIngredientVersionIngredientOptionSetsOK{}
}

/* GetIngredientVersionIngredientOptionSetsOK describes a response with status code 200, with default header values.

A paginated list of ingredient option sets
*/
type GetIngredientVersionIngredientOptionSetsOK struct {
	Payload *inventory_models.IngredientOptionSetWithUsageTypePagedList
}

func (o *GetIngredientVersionIngredientOptionSetsOK) Error() string {
	return fmt.Sprintf("[GET /v1/ingredients/{ingredient_id}/versions/{ingredient_version_id}/ingredient-option-sets][%d] getIngredientVersionIngredientOptionSetsOK  %+v", 200, o.Payload)
}
func (o *GetIngredientVersionIngredientOptionSetsOK) GetPayload() *inventory_models.IngredientOptionSetWithUsageTypePagedList {
	return o.Payload
}

func (o *GetIngredientVersionIngredientOptionSetsOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(inventory_models.IngredientOptionSetWithUsageTypePagedList)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetIngredientVersionIngredientOptionSetsDefault creates a GetIngredientVersionIngredientOptionSetsDefault with default headers values
func NewGetIngredientVersionIngredientOptionSetsDefault(code int) *GetIngredientVersionIngredientOptionSetsDefault {
	return &GetIngredientVersionIngredientOptionSetsDefault{
		_statusCode: code,
	}
}

/* GetIngredientVersionIngredientOptionSetsDefault describes a response with status code -1, with default header values.

generic error response
*/
type GetIngredientVersionIngredientOptionSetsDefault struct {
	_statusCode int

	Payload *inventory_models.RestAPIError
}

// Code gets the status code for the get ingredient version ingredient option sets default response
func (o *GetIngredientVersionIngredientOptionSetsDefault) Code() int {
	return o._statusCode
}

func (o *GetIngredientVersionIngredientOptionSetsDefault) Error() string {
	return fmt.Sprintf("[GET /v1/ingredients/{ingredient_id}/versions/{ingredient_version_id}/ingredient-option-sets][%d] getIngredientVersionIngredientOptionSets default  %+v", o._statusCode, o.Payload)
}
func (o *GetIngredientVersionIngredientOptionSetsDefault) GetPayload() *inventory_models.RestAPIError {
	return o.Payload
}

func (o *GetIngredientVersionIngredientOptionSetsDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(inventory_models.RestAPIError)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
