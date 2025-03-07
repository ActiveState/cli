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

// GetIngredientVersionAuthorsReader is a Reader for the GetIngredientVersionAuthors structure.
type GetIngredientVersionAuthorsReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *GetIngredientVersionAuthorsReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewGetIngredientVersionAuthorsOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewGetIngredientVersionAuthorsDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewGetIngredientVersionAuthorsOK creates a GetIngredientVersionAuthorsOK with default headers values
func NewGetIngredientVersionAuthorsOK() *GetIngredientVersionAuthorsOK {
	return &GetIngredientVersionAuthorsOK{}
}

/* GetIngredientVersionAuthorsOK describes a response with status code 200, with default header values.

A paginated list of authors
*/
type GetIngredientVersionAuthorsOK struct {
	Payload *inventory_models.AuthorPagedList
}

func (o *GetIngredientVersionAuthorsOK) Error() string {
	return fmt.Sprintf("[GET /v1/ingredients/{ingredient_id}/versions/{ingredient_version_id}/authors][%d] getIngredientVersionAuthorsOK  %+v", 200, o.Payload)
}
func (o *GetIngredientVersionAuthorsOK) GetPayload() *inventory_models.AuthorPagedList {
	return o.Payload
}

func (o *GetIngredientVersionAuthorsOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(inventory_models.AuthorPagedList)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetIngredientVersionAuthorsDefault creates a GetIngredientVersionAuthorsDefault with default headers values
func NewGetIngredientVersionAuthorsDefault(code int) *GetIngredientVersionAuthorsDefault {
	return &GetIngredientVersionAuthorsDefault{
		_statusCode: code,
	}
}

/* GetIngredientVersionAuthorsDefault describes a response with status code -1, with default header values.

generic error response
*/
type GetIngredientVersionAuthorsDefault struct {
	_statusCode int

	Payload *inventory_models.RestAPIError
}

// Code gets the status code for the get ingredient version authors default response
func (o *GetIngredientVersionAuthorsDefault) Code() int {
	return o._statusCode
}

func (o *GetIngredientVersionAuthorsDefault) Error() string {
	return fmt.Sprintf("[GET /v1/ingredients/{ingredient_id}/versions/{ingredient_version_id}/authors][%d] getIngredientVersionAuthors default  %+v", o._statusCode, o.Payload)
}
func (o *GetIngredientVersionAuthorsDefault) GetPayload() *inventory_models.RestAPIError {
	return o.Payload
}

func (o *GetIngredientVersionAuthorsDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(inventory_models.RestAPIError)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
