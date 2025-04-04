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

// NewGetOperatingSystemParams creates a new GetOperatingSystemParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewGetOperatingSystemParams() *GetOperatingSystemParams {
	return &GetOperatingSystemParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewGetOperatingSystemParamsWithTimeout creates a new GetOperatingSystemParams object
// with the ability to set a timeout on a request.
func NewGetOperatingSystemParamsWithTimeout(timeout time.Duration) *GetOperatingSystemParams {
	return &GetOperatingSystemParams{
		timeout: timeout,
	}
}

// NewGetOperatingSystemParamsWithContext creates a new GetOperatingSystemParams object
// with the ability to set a context for a request.
func NewGetOperatingSystemParamsWithContext(ctx context.Context) *GetOperatingSystemParams {
	return &GetOperatingSystemParams{
		Context: ctx,
	}
}

// NewGetOperatingSystemParamsWithHTTPClient creates a new GetOperatingSystemParams object
// with the ability to set a custom HTTPClient for a request.
func NewGetOperatingSystemParamsWithHTTPClient(client *http.Client) *GetOperatingSystemParams {
	return &GetOperatingSystemParams{
		HTTPClient: client,
	}
}

/* GetOperatingSystemParams contains all the parameters to send to the API endpoint
   for the get operating system operation.

   Typically these are written to a http.Request.
*/
type GetOperatingSystemParams struct {

	// OperatingSystemID.
	//
	// Format: uuid
	OperatingSystemID strfmt.UUID

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the get operating system params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *GetOperatingSystemParams) WithDefaults() *GetOperatingSystemParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the get operating system params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *GetOperatingSystemParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the get operating system params
func (o *GetOperatingSystemParams) WithTimeout(timeout time.Duration) *GetOperatingSystemParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the get operating system params
func (o *GetOperatingSystemParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the get operating system params
func (o *GetOperatingSystemParams) WithContext(ctx context.Context) *GetOperatingSystemParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the get operating system params
func (o *GetOperatingSystemParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the get operating system params
func (o *GetOperatingSystemParams) WithHTTPClient(client *http.Client) *GetOperatingSystemParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the get operating system params
func (o *GetOperatingSystemParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithOperatingSystemID adds the operatingSystemID to the get operating system params
func (o *GetOperatingSystemParams) WithOperatingSystemID(operatingSystemID strfmt.UUID) *GetOperatingSystemParams {
	o.SetOperatingSystemID(operatingSystemID)
	return o
}

// SetOperatingSystemID adds the operatingSystemId to the get operating system params
func (o *GetOperatingSystemParams) SetOperatingSystemID(operatingSystemID strfmt.UUID) {
	o.OperatingSystemID = operatingSystemID
}

// WriteToRequest writes these params to a swagger request
func (o *GetOperatingSystemParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param operating_system_id
	if err := r.SetPathParam("operating_system_id", o.OperatingSystemID.String()); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
