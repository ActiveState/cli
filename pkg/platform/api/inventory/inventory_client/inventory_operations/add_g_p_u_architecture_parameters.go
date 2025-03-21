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

	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
)

// NewAddGPUArchitectureParams creates a new AddGPUArchitectureParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewAddGPUArchitectureParams() *AddGPUArchitectureParams {
	return &AddGPUArchitectureParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewAddGPUArchitectureParamsWithTimeout creates a new AddGPUArchitectureParams object
// with the ability to set a timeout on a request.
func NewAddGPUArchitectureParamsWithTimeout(timeout time.Duration) *AddGPUArchitectureParams {
	return &AddGPUArchitectureParams{
		timeout: timeout,
	}
}

// NewAddGPUArchitectureParamsWithContext creates a new AddGPUArchitectureParams object
// with the ability to set a context for a request.
func NewAddGPUArchitectureParamsWithContext(ctx context.Context) *AddGPUArchitectureParams {
	return &AddGPUArchitectureParams{
		Context: ctx,
	}
}

// NewAddGPUArchitectureParamsWithHTTPClient creates a new AddGPUArchitectureParams object
// with the ability to set a custom HTTPClient for a request.
func NewAddGPUArchitectureParamsWithHTTPClient(client *http.Client) *AddGPUArchitectureParams {
	return &AddGPUArchitectureParams{
		HTTPClient: client,
	}
}

/* AddGPUArchitectureParams contains all the parameters to send to the API endpoint
   for the add g p u architecture operation.

   Typically these are written to a http.Request.
*/
type AddGPUArchitectureParams struct {

	// GpuArchitecture.
	GpuArchitecture *inventory_models.GpuArchitectureCore

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the add g p u architecture params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *AddGPUArchitectureParams) WithDefaults() *AddGPUArchitectureParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the add g p u architecture params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *AddGPUArchitectureParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the add g p u architecture params
func (o *AddGPUArchitectureParams) WithTimeout(timeout time.Duration) *AddGPUArchitectureParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the add g p u architecture params
func (o *AddGPUArchitectureParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the add g p u architecture params
func (o *AddGPUArchitectureParams) WithContext(ctx context.Context) *AddGPUArchitectureParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the add g p u architecture params
func (o *AddGPUArchitectureParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the add g p u architecture params
func (o *AddGPUArchitectureParams) WithHTTPClient(client *http.Client) *AddGPUArchitectureParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the add g p u architecture params
func (o *AddGPUArchitectureParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithGpuArchitecture adds the gpuArchitecture to the add g p u architecture params
func (o *AddGPUArchitectureParams) WithGpuArchitecture(gpuArchitecture *inventory_models.GpuArchitectureCore) *AddGPUArchitectureParams {
	o.SetGpuArchitecture(gpuArchitecture)
	return o
}

// SetGpuArchitecture adds the gpuArchitecture to the add g p u architecture params
func (o *AddGPUArchitectureParams) SetGpuArchitecture(gpuArchitecture *inventory_models.GpuArchitectureCore) {
	o.GpuArchitecture = gpuArchitecture
}

// WriteToRequest writes these params to a swagger request
func (o *AddGPUArchitectureParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error
	if o.GpuArchitecture != nil {
		if err := r.SetBodyParam(o.GpuArchitecture); err != nil {
			return err
		}
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
