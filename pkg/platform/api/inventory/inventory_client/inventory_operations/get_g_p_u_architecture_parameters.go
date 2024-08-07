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

// NewGetGPUArchitectureParams creates a new GetGPUArchitectureParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewGetGPUArchitectureParams() *GetGPUArchitectureParams {
	return &GetGPUArchitectureParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewGetGPUArchitectureParamsWithTimeout creates a new GetGPUArchitectureParams object
// with the ability to set a timeout on a request.
func NewGetGPUArchitectureParamsWithTimeout(timeout time.Duration) *GetGPUArchitectureParams {
	return &GetGPUArchitectureParams{
		timeout: timeout,
	}
}

// NewGetGPUArchitectureParamsWithContext creates a new GetGPUArchitectureParams object
// with the ability to set a context for a request.
func NewGetGPUArchitectureParamsWithContext(ctx context.Context) *GetGPUArchitectureParams {
	return &GetGPUArchitectureParams{
		Context: ctx,
	}
}

// NewGetGPUArchitectureParamsWithHTTPClient creates a new GetGPUArchitectureParams object
// with the ability to set a custom HTTPClient for a request.
func NewGetGPUArchitectureParamsWithHTTPClient(client *http.Client) *GetGPUArchitectureParams {
	return &GetGPUArchitectureParams{
		HTTPClient: client,
	}
}

/* GetGPUArchitectureParams contains all the parameters to send to the API endpoint
   for the get g p u architecture operation.

   Typically these are written to a http.Request.
*/
type GetGPUArchitectureParams struct {

	/* AllowUnstable.

	   Whether to show an unstable revision of a resource if there is an available unstable version newer than the newest available stable version
	*/
	AllowUnstable *bool

	// GpuArchitectureID.
	//
	// Format: uuid
	GpuArchitectureID strfmt.UUID

	/* StateAt.

	   Show the state of a resource as it was at the specified timestamp. If omitted, shows the current state of the resource.

	   Format: date-time
	*/
	StateAt *strfmt.DateTime

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the get g p u architecture params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *GetGPUArchitectureParams) WithDefaults() *GetGPUArchitectureParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the get g p u architecture params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *GetGPUArchitectureParams) SetDefaults() {
	var (
		allowUnstableDefault = bool(false)
	)

	val := GetGPUArchitectureParams{
		AllowUnstable: &allowUnstableDefault,
	}

	val.timeout = o.timeout
	val.Context = o.Context
	val.HTTPClient = o.HTTPClient
	*o = val
}

// WithTimeout adds the timeout to the get g p u architecture params
func (o *GetGPUArchitectureParams) WithTimeout(timeout time.Duration) *GetGPUArchitectureParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the get g p u architecture params
func (o *GetGPUArchitectureParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the get g p u architecture params
func (o *GetGPUArchitectureParams) WithContext(ctx context.Context) *GetGPUArchitectureParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the get g p u architecture params
func (o *GetGPUArchitectureParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the get g p u architecture params
func (o *GetGPUArchitectureParams) WithHTTPClient(client *http.Client) *GetGPUArchitectureParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the get g p u architecture params
func (o *GetGPUArchitectureParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithAllowUnstable adds the allowUnstable to the get g p u architecture params
func (o *GetGPUArchitectureParams) WithAllowUnstable(allowUnstable *bool) *GetGPUArchitectureParams {
	o.SetAllowUnstable(allowUnstable)
	return o
}

// SetAllowUnstable adds the allowUnstable to the get g p u architecture params
func (o *GetGPUArchitectureParams) SetAllowUnstable(allowUnstable *bool) {
	o.AllowUnstable = allowUnstable
}

// WithGpuArchitectureID adds the gpuArchitectureID to the get g p u architecture params
func (o *GetGPUArchitectureParams) WithGpuArchitectureID(gpuArchitectureID strfmt.UUID) *GetGPUArchitectureParams {
	o.SetGpuArchitectureID(gpuArchitectureID)
	return o
}

// SetGpuArchitectureID adds the gpuArchitectureId to the get g p u architecture params
func (o *GetGPUArchitectureParams) SetGpuArchitectureID(gpuArchitectureID strfmt.UUID) {
	o.GpuArchitectureID = gpuArchitectureID
}

// WithStateAt adds the stateAt to the get g p u architecture params
func (o *GetGPUArchitectureParams) WithStateAt(stateAt *strfmt.DateTime) *GetGPUArchitectureParams {
	o.SetStateAt(stateAt)
	return o
}

// SetStateAt adds the stateAt to the get g p u architecture params
func (o *GetGPUArchitectureParams) SetStateAt(stateAt *strfmt.DateTime) {
	o.StateAt = stateAt
}

// WriteToRequest writes these params to a swagger request
func (o *GetGPUArchitectureParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if o.AllowUnstable != nil {

		// query param allow_unstable
		var qrAllowUnstable bool

		if o.AllowUnstable != nil {
			qrAllowUnstable = *o.AllowUnstable
		}
		qAllowUnstable := swag.FormatBool(qrAllowUnstable)
		if qAllowUnstable != "" {

			if err := r.SetQueryParam("allow_unstable", qAllowUnstable); err != nil {
				return err
			}
		}
	}

	// path param gpu_architecture_id
	if err := r.SetPathParam("gpu_architecture_id", o.GpuArchitectureID.String()); err != nil {
		return err
	}

	if o.StateAt != nil {

		// query param state_at
		var qrStateAt strfmt.DateTime

		if o.StateAt != nil {
			qrStateAt = *o.StateAt
		}
		qStateAt := qrStateAt.String()
		if qStateAt != "" {

			if err := r.SetQueryParam("state_at", qStateAt); err != nil {
				return err
			}
		}
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
