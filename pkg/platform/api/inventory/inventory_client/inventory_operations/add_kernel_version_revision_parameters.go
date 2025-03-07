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

// NewAddKernelVersionRevisionParams creates a new AddKernelVersionRevisionParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewAddKernelVersionRevisionParams() *AddKernelVersionRevisionParams {
	return &AddKernelVersionRevisionParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewAddKernelVersionRevisionParamsWithTimeout creates a new AddKernelVersionRevisionParams object
// with the ability to set a timeout on a request.
func NewAddKernelVersionRevisionParamsWithTimeout(timeout time.Duration) *AddKernelVersionRevisionParams {
	return &AddKernelVersionRevisionParams{
		timeout: timeout,
	}
}

// NewAddKernelVersionRevisionParamsWithContext creates a new AddKernelVersionRevisionParams object
// with the ability to set a context for a request.
func NewAddKernelVersionRevisionParamsWithContext(ctx context.Context) *AddKernelVersionRevisionParams {
	return &AddKernelVersionRevisionParams{
		Context: ctx,
	}
}

// NewAddKernelVersionRevisionParamsWithHTTPClient creates a new AddKernelVersionRevisionParams object
// with the ability to set a custom HTTPClient for a request.
func NewAddKernelVersionRevisionParamsWithHTTPClient(client *http.Client) *AddKernelVersionRevisionParams {
	return &AddKernelVersionRevisionParams{
		HTTPClient: client,
	}
}

/* AddKernelVersionRevisionParams contains all the parameters to send to the API endpoint
   for the add kernel version revision operation.

   Typically these are written to a http.Request.
*/
type AddKernelVersionRevisionParams struct {

	// KernelID.
	//
	// Format: uuid
	KernelID strfmt.UUID

	// KernelVersionID.
	//
	// Format: uuid
	KernelVersionID strfmt.UUID

	// KernelVersionRevision.
	KernelVersionRevision *inventory_models.RevisionedFeatureProvider

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the add kernel version revision params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *AddKernelVersionRevisionParams) WithDefaults() *AddKernelVersionRevisionParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the add kernel version revision params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *AddKernelVersionRevisionParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the add kernel version revision params
func (o *AddKernelVersionRevisionParams) WithTimeout(timeout time.Duration) *AddKernelVersionRevisionParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the add kernel version revision params
func (o *AddKernelVersionRevisionParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the add kernel version revision params
func (o *AddKernelVersionRevisionParams) WithContext(ctx context.Context) *AddKernelVersionRevisionParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the add kernel version revision params
func (o *AddKernelVersionRevisionParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the add kernel version revision params
func (o *AddKernelVersionRevisionParams) WithHTTPClient(client *http.Client) *AddKernelVersionRevisionParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the add kernel version revision params
func (o *AddKernelVersionRevisionParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithKernelID adds the kernelID to the add kernel version revision params
func (o *AddKernelVersionRevisionParams) WithKernelID(kernelID strfmt.UUID) *AddKernelVersionRevisionParams {
	o.SetKernelID(kernelID)
	return o
}

// SetKernelID adds the kernelId to the add kernel version revision params
func (o *AddKernelVersionRevisionParams) SetKernelID(kernelID strfmt.UUID) {
	o.KernelID = kernelID
}

// WithKernelVersionID adds the kernelVersionID to the add kernel version revision params
func (o *AddKernelVersionRevisionParams) WithKernelVersionID(kernelVersionID strfmt.UUID) *AddKernelVersionRevisionParams {
	o.SetKernelVersionID(kernelVersionID)
	return o
}

// SetKernelVersionID adds the kernelVersionId to the add kernel version revision params
func (o *AddKernelVersionRevisionParams) SetKernelVersionID(kernelVersionID strfmt.UUID) {
	o.KernelVersionID = kernelVersionID
}

// WithKernelVersionRevision adds the kernelVersionRevision to the add kernel version revision params
func (o *AddKernelVersionRevisionParams) WithKernelVersionRevision(kernelVersionRevision *inventory_models.RevisionedFeatureProvider) *AddKernelVersionRevisionParams {
	o.SetKernelVersionRevision(kernelVersionRevision)
	return o
}

// SetKernelVersionRevision adds the kernelVersionRevision to the add kernel version revision params
func (o *AddKernelVersionRevisionParams) SetKernelVersionRevision(kernelVersionRevision *inventory_models.RevisionedFeatureProvider) {
	o.KernelVersionRevision = kernelVersionRevision
}

// WriteToRequest writes these params to a swagger request
func (o *AddKernelVersionRevisionParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param kernel_id
	if err := r.SetPathParam("kernel_id", o.KernelID.String()); err != nil {
		return err
	}

	// path param kernel_version_id
	if err := r.SetPathParam("kernel_version_id", o.KernelVersionID.String()); err != nil {
		return err
	}
	if o.KernelVersionRevision != nil {
		if err := r.SetBodyParam(o.KernelVersionRevision); err != nil {
			return err
		}
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
