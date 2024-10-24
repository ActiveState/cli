// Code generated by go-swagger; DO NOT EDIT.

package oauth

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

// NewAuthDevicePutParams creates a new AuthDevicePutParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewAuthDevicePutParams() *AuthDevicePutParams {
	return &AuthDevicePutParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewAuthDevicePutParamsWithTimeout creates a new AuthDevicePutParams object
// with the ability to set a timeout on a request.
func NewAuthDevicePutParamsWithTimeout(timeout time.Duration) *AuthDevicePutParams {
	return &AuthDevicePutParams{
		timeout: timeout,
	}
}

// NewAuthDevicePutParamsWithContext creates a new AuthDevicePutParams object
// with the ability to set a context for a request.
func NewAuthDevicePutParamsWithContext(ctx context.Context) *AuthDevicePutParams {
	return &AuthDevicePutParams{
		Context: ctx,
	}
}

// NewAuthDevicePutParamsWithHTTPClient creates a new AuthDevicePutParams object
// with the ability to set a custom HTTPClient for a request.
func NewAuthDevicePutParamsWithHTTPClient(client *http.Client) *AuthDevicePutParams {
	return &AuthDevicePutParams{
		HTTPClient: client,
	}
}

/* AuthDevicePutParams contains all the parameters to send to the API endpoint
   for the auth device put operation.

   Typically these are written to a http.Request.
*/
type AuthDevicePutParams struct {

	/* UserCode.

	   userCode received from /oauth/authorize/device:POST

	   Format: uuid
	*/
	UserCode strfmt.UUID

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the auth device put params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *AuthDevicePutParams) WithDefaults() *AuthDevicePutParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the auth device put params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *AuthDevicePutParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the auth device put params
func (o *AuthDevicePutParams) WithTimeout(timeout time.Duration) *AuthDevicePutParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the auth device put params
func (o *AuthDevicePutParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the auth device put params
func (o *AuthDevicePutParams) WithContext(ctx context.Context) *AuthDevicePutParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the auth device put params
func (o *AuthDevicePutParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the auth device put params
func (o *AuthDevicePutParams) WithHTTPClient(client *http.Client) *AuthDevicePutParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the auth device put params
func (o *AuthDevicePutParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithUserCode adds the userCode to the auth device put params
func (o *AuthDevicePutParams) WithUserCode(userCode strfmt.UUID) *AuthDevicePutParams {
	o.SetUserCode(userCode)
	return o
}

// SetUserCode adds the userCode to the auth device put params
func (o *AuthDevicePutParams) SetUserCode(userCode strfmt.UUID) {
	o.UserCode = userCode
}

// WriteToRequest writes these params to a swagger request
func (o *AuthDevicePutParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// query param user_code
	qrUserCode := o.UserCode
	qUserCode := qrUserCode.String()
	if qUserCode != "" {

		if err := r.SetQueryParam("user_code", qUserCode); err != nil {
			return err
		}
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
