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
	"github.com/go-openapi/swag"
)

// NewGetBuildScriptsParams creates a new GetBuildScriptsParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewGetBuildScriptsParams() *GetBuildScriptsParams {
	return &GetBuildScriptsParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewGetBuildScriptsParamsWithTimeout creates a new GetBuildScriptsParams object
// with the ability to set a timeout on a request.
func NewGetBuildScriptsParamsWithTimeout(timeout time.Duration) *GetBuildScriptsParams {
	return &GetBuildScriptsParams{
		timeout: timeout,
	}
}

// NewGetBuildScriptsParamsWithContext creates a new GetBuildScriptsParams object
// with the ability to set a context for a request.
func NewGetBuildScriptsParamsWithContext(ctx context.Context) *GetBuildScriptsParams {
	return &GetBuildScriptsParams{
		Context: ctx,
	}
}

// NewGetBuildScriptsParamsWithHTTPClient creates a new GetBuildScriptsParams object
// with the ability to set a custom HTTPClient for a request.
func NewGetBuildScriptsParamsWithHTTPClient(client *http.Client) *GetBuildScriptsParams {
	return &GetBuildScriptsParams{
		HTTPClient: client,
	}
}

/* GetBuildScriptsParams contains all the parameters to send to the API endpoint
   for the get build scripts operation.

   Typically these are written to a http.Request.
*/
type GetBuildScriptsParams struct {

	/* Limit.

	   The maximum number of items returned per page

	   Default: 50
	*/
	Limit *int64

	/* Page.

	   The page number returned

	   Default: 1
	*/
	Page *int64

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the get build scripts params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *GetBuildScriptsParams) WithDefaults() *GetBuildScriptsParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the get build scripts params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *GetBuildScriptsParams) SetDefaults() {
	var (
		limitDefault = int64(50)

		pageDefault = int64(1)
	)

	val := GetBuildScriptsParams{
		Limit: &limitDefault,
		Page:  &pageDefault,
	}

	val.timeout = o.timeout
	val.Context = o.Context
	val.HTTPClient = o.HTTPClient
	*o = val
}

// WithTimeout adds the timeout to the get build scripts params
func (o *GetBuildScriptsParams) WithTimeout(timeout time.Duration) *GetBuildScriptsParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the get build scripts params
func (o *GetBuildScriptsParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the get build scripts params
func (o *GetBuildScriptsParams) WithContext(ctx context.Context) *GetBuildScriptsParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the get build scripts params
func (o *GetBuildScriptsParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the get build scripts params
func (o *GetBuildScriptsParams) WithHTTPClient(client *http.Client) *GetBuildScriptsParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the get build scripts params
func (o *GetBuildScriptsParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithLimit adds the limit to the get build scripts params
func (o *GetBuildScriptsParams) WithLimit(limit *int64) *GetBuildScriptsParams {
	o.SetLimit(limit)
	return o
}

// SetLimit adds the limit to the get build scripts params
func (o *GetBuildScriptsParams) SetLimit(limit *int64) {
	o.Limit = limit
}

// WithPage adds the page to the get build scripts params
func (o *GetBuildScriptsParams) WithPage(page *int64) *GetBuildScriptsParams {
	o.SetPage(page)
	return o
}

// SetPage adds the page to the get build scripts params
func (o *GetBuildScriptsParams) SetPage(page *int64) {
	o.Page = page
}

// WriteToRequest writes these params to a swagger request
func (o *GetBuildScriptsParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if o.Limit != nil {

		// query param limit
		var qrLimit int64

		if o.Limit != nil {
			qrLimit = *o.Limit
		}
		qLimit := swag.FormatInt64(qrLimit)
		if qLimit != "" {

			if err := r.SetQueryParam("limit", qLimit); err != nil {
				return err
			}
		}
	}

	if o.Page != nil {

		// query param page
		var qrPage int64

		if o.Page != nil {
			qrPage = *o.Page
		}
		qPage := swag.FormatInt64(qrPage)
		if qPage != "" {

			if err := r.SetQueryParam("page", qPage); err != nil {
				return err
			}
		}
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
