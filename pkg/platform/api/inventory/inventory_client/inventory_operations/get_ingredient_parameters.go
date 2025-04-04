// Code generated by go-swagger; DO NOT EDIT.

package inventory_operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"net/http"
	"time"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	cr "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
)

// NewGetIngredientParams creates a new GetIngredientParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewGetIngredientParams() *GetIngredientParams {
	return &GetIngredientParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewGetIngredientParamsWithTimeout creates a new GetIngredientParams object
// with the ability to set a timeout on a request.
func NewGetIngredientParamsWithTimeout(timeout time.Duration) *GetIngredientParams {
	return &GetIngredientParams{
		timeout: timeout,
	}
}

// NewGetIngredientParamsWithContext creates a new GetIngredientParams object
// with the ability to set a context for a request.
func NewGetIngredientParamsWithContext(ctx context.Context) *GetIngredientParams {
	return &GetIngredientParams{
		Context: ctx,
	}
}

// NewGetIngredientParamsWithHTTPClient creates a new GetIngredientParams object
// with the ability to set a custom HTTPClient for a request.
func NewGetIngredientParamsWithHTTPClient(client *http.Client) *GetIngredientParams {
	return &GetIngredientParams{
		HTTPClient: client,
	}
}

/* GetIngredientParams contains all the parameters to send to the API endpoint
   for the get ingredient operation.

   Typically these are written to a http.Request.
*/
type GetIngredientParams struct {

	// IngredientID.
	//
	// Format: uuid
	IngredientID strfmt.UUID

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the get ingredient params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *GetIngredientParams) WithDefaults() *GetIngredientParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the get ingredient params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *GetIngredientParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the get ingredient params
func (o *GetIngredientParams) WithTimeout(timeout time.Duration) *GetIngredientParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the get ingredient params
func (o *GetIngredientParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the get ingredient params
func (o *GetIngredientParams) WithContext(ctx context.Context) *GetIngredientParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the get ingredient params
func (o *GetIngredientParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the get ingredient params
func (o *GetIngredientParams) WithHTTPClient(client *http.Client) *GetIngredientParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the get ingredient params
func (o *GetIngredientParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithIngredientID adds the ingredientID to the get ingredient params
func (o *GetIngredientParams) WithIngredientID(ingredientID strfmt.UUID) *GetIngredientParams {
	o.SetIngredientID(ingredientID)
	return o
}

// SetIngredientID adds the ingredientId to the get ingredient params
func (o *GetIngredientParams) SetIngredientID(ingredientID strfmt.UUID) {
	o.IngredientID = ingredientID
}

// WriteToRequest writes these params to a swagger request
func (o *GetIngredientParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param ingredient_id
	if err := r.SetPathParam("ingredient_id", o.IngredientID.String()); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
