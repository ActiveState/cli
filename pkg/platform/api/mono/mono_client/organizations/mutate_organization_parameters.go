// Code generated by go-swagger; DO NOT EDIT.

package organizations

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

	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
)

// NewMutateOrganizationParams creates a new MutateOrganizationParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewMutateOrganizationParams() *MutateOrganizationParams {
	return &MutateOrganizationParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewMutateOrganizationParamsWithTimeout creates a new MutateOrganizationParams object
// with the ability to set a timeout on a request.
func NewMutateOrganizationParamsWithTimeout(timeout time.Duration) *MutateOrganizationParams {
	return &MutateOrganizationParams{
		timeout: timeout,
	}
}

// NewMutateOrganizationParamsWithContext creates a new MutateOrganizationParams object
// with the ability to set a context for a request.
func NewMutateOrganizationParamsWithContext(ctx context.Context) *MutateOrganizationParams {
	return &MutateOrganizationParams{
		Context: ctx,
	}
}

// NewMutateOrganizationParamsWithHTTPClient creates a new MutateOrganizationParams object
// with the ability to set a custom HTTPClient for a request.
func NewMutateOrganizationParamsWithHTTPClient(client *http.Client) *MutateOrganizationParams {
	return &MutateOrganizationParams{
		HTTPClient: client,
	}
}

/* MutateOrganizationParams contains all the parameters to send to the API endpoint
   for the mutate organization operation.

   Typically these are written to a http.Request.
*/
type MutateOrganizationParams struct {

	/* IdentifierType.

	   what kind of thing the provided organizationIdentifier is

	   Default: "URLname"
	*/
	IdentifierType *string

	// Mutation.
	Mutation *mono_models.OrganizationMutationEditable

	/* OrganizationIdentifier.

	   identifier (URLname, by default) of the desired organization
	*/
	OrganizationIdentifier string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the mutate organization params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *MutateOrganizationParams) WithDefaults() *MutateOrganizationParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the mutate organization params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *MutateOrganizationParams) SetDefaults() {
	var (
		identifierTypeDefault = string("URLname")
	)

	val := MutateOrganizationParams{
		IdentifierType: &identifierTypeDefault,
	}

	val.timeout = o.timeout
	val.Context = o.Context
	val.HTTPClient = o.HTTPClient
	*o = val
}

// WithTimeout adds the timeout to the mutate organization params
func (o *MutateOrganizationParams) WithTimeout(timeout time.Duration) *MutateOrganizationParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the mutate organization params
func (o *MutateOrganizationParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the mutate organization params
func (o *MutateOrganizationParams) WithContext(ctx context.Context) *MutateOrganizationParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the mutate organization params
func (o *MutateOrganizationParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the mutate organization params
func (o *MutateOrganizationParams) WithHTTPClient(client *http.Client) *MutateOrganizationParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the mutate organization params
func (o *MutateOrganizationParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithIdentifierType adds the identifierType to the mutate organization params
func (o *MutateOrganizationParams) WithIdentifierType(identifierType *string) *MutateOrganizationParams {
	o.SetIdentifierType(identifierType)
	return o
}

// SetIdentifierType adds the identifierType to the mutate organization params
func (o *MutateOrganizationParams) SetIdentifierType(identifierType *string) {
	o.IdentifierType = identifierType
}

// WithMutation adds the mutation to the mutate organization params
func (o *MutateOrganizationParams) WithMutation(mutation *mono_models.OrganizationMutationEditable) *MutateOrganizationParams {
	o.SetMutation(mutation)
	return o
}

// SetMutation adds the mutation to the mutate organization params
func (o *MutateOrganizationParams) SetMutation(mutation *mono_models.OrganizationMutationEditable) {
	o.Mutation = mutation
}

// WithOrganizationIdentifier adds the organizationIdentifier to the mutate organization params
func (o *MutateOrganizationParams) WithOrganizationIdentifier(organizationIdentifier string) *MutateOrganizationParams {
	o.SetOrganizationIdentifier(organizationIdentifier)
	return o
}

// SetOrganizationIdentifier adds the organizationIdentifier to the mutate organization params
func (o *MutateOrganizationParams) SetOrganizationIdentifier(organizationIdentifier string) {
	o.OrganizationIdentifier = organizationIdentifier
}

// WriteToRequest writes these params to a swagger request
func (o *MutateOrganizationParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if o.IdentifierType != nil {

		// query param identifierType
		var qrIdentifierType string

		if o.IdentifierType != nil {
			qrIdentifierType = *o.IdentifierType
		}
		qIdentifierType := qrIdentifierType
		if qIdentifierType != "" {

			if err := r.SetQueryParam("identifierType", qIdentifierType); err != nil {
				return err
			}
		}
	}
	if o.Mutation != nil {
		if err := r.SetBodyParam(o.Mutation); err != nil {
			return err
		}
	}

	// path param organizationIdentifier
	if err := r.SetPathParam("organizationIdentifier", o.OrganizationIdentifier); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
