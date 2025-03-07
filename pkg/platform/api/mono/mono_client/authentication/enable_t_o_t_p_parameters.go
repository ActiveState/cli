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

// NewEnableTOTPParams creates a new EnableTOTPParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewEnableTOTPParams() *EnableTOTPParams {
	return &EnableTOTPParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewEnableTOTPParamsWithTimeout creates a new EnableTOTPParams object
// with the ability to set a timeout on a request.
func NewEnableTOTPParamsWithTimeout(timeout time.Duration) *EnableTOTPParams {
	return &EnableTOTPParams{
		timeout: timeout,
	}
}

// NewEnableTOTPParamsWithContext creates a new EnableTOTPParams object
// with the ability to set a context for a request.
func NewEnableTOTPParamsWithContext(ctx context.Context) *EnableTOTPParams {
	return &EnableTOTPParams{
		Context: ctx,
	}
}

// NewEnableTOTPParamsWithHTTPClient creates a new EnableTOTPParams object
// with the ability to set a custom HTTPClient for a request.
func NewEnableTOTPParamsWithHTTPClient(client *http.Client) *EnableTOTPParams {
	return &EnableTOTPParams{
		HTTPClient: client,
	}
}

/* EnableTOTPParams contains all the parameters to send to the API endpoint
   for the enable t o t p operation.

   Typically these are written to a http.Request.
*/
type EnableTOTPParams struct {

	/* Code.

	   TOTP 2FA Rolling Code
	*/
	Code string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the enable t o t p params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *EnableTOTPParams) WithDefaults() *EnableTOTPParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the enable t o t p params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *EnableTOTPParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the enable t o t p params
func (o *EnableTOTPParams) WithTimeout(timeout time.Duration) *EnableTOTPParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the enable t o t p params
func (o *EnableTOTPParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the enable t o t p params
func (o *EnableTOTPParams) WithContext(ctx context.Context) *EnableTOTPParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the enable t o t p params
func (o *EnableTOTPParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the enable t o t p params
func (o *EnableTOTPParams) WithHTTPClient(client *http.Client) *EnableTOTPParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the enable t o t p params
func (o *EnableTOTPParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithCode adds the code to the enable t o t p params
func (o *EnableTOTPParams) WithCode(code string) *EnableTOTPParams {
	o.SetCode(code)
	return o
}

// SetCode adds the code to the enable t o t p params
func (o *EnableTOTPParams) SetCode(code string) {
	o.Code = code
}

// WriteToRequest writes these params to a swagger request
func (o *EnableTOTPParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// query param code
	qrCode := o.Code
	qCode := qrCode
	if qCode != "" {

		if err := r.SetQueryParam("code", qCode); err != nil {
			return err
		}
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
