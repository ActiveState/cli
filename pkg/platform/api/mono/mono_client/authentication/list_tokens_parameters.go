// Code generated by go-swagger; DO NOT EDIT.

package authentication

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

// NewListTokensParams creates a new ListTokensParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewListTokensParams() *ListTokensParams {
	return &ListTokensParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewListTokensParamsWithTimeout creates a new ListTokensParams object
// with the ability to set a timeout on a request.
func NewListTokensParamsWithTimeout(timeout time.Duration) *ListTokensParams {
	return &ListTokensParams{
		timeout: timeout,
	}
}

// NewListTokensParamsWithContext creates a new ListTokensParams object
// with the ability to set a context for a request.
func NewListTokensParamsWithContext(ctx context.Context) *ListTokensParams {
	return &ListTokensParams{
		Context: ctx,
	}
}

// NewListTokensParamsWithHTTPClient creates a new ListTokensParams object
// with the ability to set a custom HTTPClient for a request.
func NewListTokensParamsWithHTTPClient(client *http.Client) *ListTokensParams {
	return &ListTokensParams{
		HTTPClient: client,
	}
}

/* ListTokensParams contains all the parameters to send to the API endpoint
   for the list tokens operation.

   Typically these are written to a http.Request.
*/
type ListTokensParams struct {
	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the list tokens params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListTokensParams) WithDefaults() *ListTokensParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the list tokens params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListTokensParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the list tokens params
func (o *ListTokensParams) WithTimeout(timeout time.Duration) *ListTokensParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list tokens params
func (o *ListTokensParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list tokens params
func (o *ListTokensParams) WithContext(ctx context.Context) *ListTokensParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list tokens params
func (o *ListTokensParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list tokens params
func (o *ListTokensParams) WithHTTPClient(client *http.Client) *ListTokensParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list tokens params
func (o *ListTokensParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WriteToRequest writes these params to a swagger request
func (o *ListTokensParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
