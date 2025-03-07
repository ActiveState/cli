// Code generated by go-swagger; DO NOT EDIT.

package projects

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
)

// New creates a new projects API client.
func New(transport runtime.ClientTransport, formats strfmt.Registry) ClientService {
	return &Client{transport: transport, formats: formats}
}

/*
Client for projects API
*/
type Client struct {
	transport runtime.ClientTransport
	formats   strfmt.Registry
}

// ClientOption is the option for Client methods
type ClientOption func(*runtime.ClientOperation)

// ClientService is the interface for Client methods
type ClientService interface {
	AddBranch(params *AddBranchParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*AddBranchOK, error)

	AddProject(params *AddProjectParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*AddProjectOK, error)

	DeleteProject(params *DeleteProjectParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*DeleteProjectOK, error)

	EditProject(params *EditProjectParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*EditProjectOK, error)

	ForkProject(params *ForkProjectParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ForkProjectOK, error)

	GetProject(params *GetProjectParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*GetProjectOK, error)

	GetProjectByID(params *GetProjectByIDParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*GetProjectByIDOK, error)

	ListProjects(params *ListProjectsParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListProjectsOK, error)

	MoveProject(params *MoveProjectParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*MoveProjectOK, error)

	SetTransport(transport runtime.ClientTransport)
}

/*
  AddBranch adds branch

  Add a branch on the specified project
*/
func (a *Client) AddBranch(params *AddBranchParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*AddBranchOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewAddBranchParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "addBranch",
		Method:             "POST",
		PathPattern:        "/projects/{projectID}/branches",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &AddBranchReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*AddBranchOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	// safeguard: normally, absent a default response, unknown success responses return an error above: so this is a codegen issue
	msg := fmt.Sprintf("unexpected success response for addBranch: API contract not enforced by server. Client expected to get an error, but got: %T", result)
	panic(msg)
}

/*
  AddProject creates a project

  Add a new project to an organization
*/
func (a *Client) AddProject(params *AddProjectParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*AddProjectOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewAddProjectParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "addProject",
		Method:             "POST",
		PathPattern:        "/organizations/{organizationName}/projects",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &AddProjectReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*AddProjectOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	// safeguard: normally, absent a default response, unknown success responses return an error above: so this is a codegen issue
	msg := fmt.Sprintf("unexpected success response for addProject: API contract not enforced by server. Client expected to get an error, but got: %T", result)
	panic(msg)
}

/*
  DeleteProject deletes a project

  Delete a Project
*/
func (a *Client) DeleteProject(params *DeleteProjectParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*DeleteProjectOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewDeleteProjectParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "deleteProject",
		Method:             "DELETE",
		PathPattern:        "/organizations/{organizationName}/projects/{projectName}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &DeleteProjectReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*DeleteProjectOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	// safeguard: normally, absent a default response, unknown success responses return an error above: so this is a codegen issue
	msg := fmt.Sprintf("unexpected success response for deleteProject: API contract not enforced by server. Client expected to get an error, but got: %T", result)
	panic(msg)
}

/*
  EditProject edits a project

  Edit a project
*/
func (a *Client) EditProject(params *EditProjectParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*EditProjectOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewEditProjectParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "editProject",
		Method:             "POST",
		PathPattern:        "/organizations/{organizationName}/projects/{projectName}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &EditProjectReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*EditProjectOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	// safeguard: normally, absent a default response, unknown success responses return an error above: so this is a codegen issue
	msg := fmt.Sprintf("unexpected success response for editProject: API contract not enforced by server. Client expected to get an error, but got: %T", result)
	panic(msg)
}

/*
  ForkProject forks a project

  Fork a project
*/
func (a *Client) ForkProject(params *ForkProjectParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ForkProjectOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewForkProjectParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "forkProject",
		Method:             "POST",
		PathPattern:        "/organizations/{organizationName}/projects/{projectName}/fork",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &ForkProjectReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*ForkProjectOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	// safeguard: normally, absent a default response, unknown success responses return an error above: so this is a codegen issue
	msg := fmt.Sprintf("unexpected success response for forkProject: API contract not enforced by server. Client expected to get an error, but got: %T", result)
	panic(msg)
}

/*
  GetProject organizations project info

  Get project details
*/
func (a *Client) GetProject(params *GetProjectParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*GetProjectOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewGetProjectParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "getProject",
		Method:             "GET",
		PathPattern:        "/organizations/{organizationName}/projects/{projectName}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &GetProjectReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*GetProjectOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	// safeguard: normally, absent a default response, unknown success responses return an error above: so this is a codegen issue
	msg := fmt.Sprintf("unexpected success response for getProject: API contract not enforced by server. Client expected to get an error, but got: %T", result)
	panic(msg)
}

/*
  GetProjectByID projects info

  Get project details by ID
*/
func (a *Client) GetProjectByID(params *GetProjectByIDParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*GetProjectByIDOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewGetProjectByIDParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "getProjectByID",
		Method:             "GET",
		PathPattern:        "/projects/{projectID}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &GetProjectByIDReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*GetProjectByIDOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	// safeguard: normally, absent a default response, unknown success responses return an error above: so this is a codegen issue
	msg := fmt.Sprintf("unexpected success response for getProjectByID: API contract not enforced by server. Client expected to get an error, but got: %T", result)
	panic(msg)
}

/*
  ListProjects organizations projects

  Return a list of projects for an organization
*/
func (a *Client) ListProjects(params *ListProjectsParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListProjectsOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListProjectsParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "listProjects",
		Method:             "GET",
		PathPattern:        "/organizations/{organizationName}/projects",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &ListProjectsReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*ListProjectsOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	// safeguard: normally, absent a default response, unknown success responses return an error above: so this is a codegen issue
	msg := fmt.Sprintf("unexpected success response for listProjects: API contract not enforced by server. Client expected to get an error, but got: %T", result)
	panic(msg)
}

/*
  MoveProject moves a project to a different organization

  Move a project to a different organization
*/
func (a *Client) MoveProject(params *MoveProjectParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*MoveProjectOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewMoveProjectParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "moveProject",
		Method:             "POST",
		PathPattern:        "/organizations/{organizationIdentifier}/projects/{projectName}/move",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &MoveProjectReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*MoveProjectOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	// safeguard: normally, absent a default response, unknown success responses return an error above: so this is a codegen issue
	msg := fmt.Sprintf("unexpected success response for moveProject: API contract not enforced by server. Client expected to get an error, but got: %T", result)
	panic(msg)
}

// SetTransport changes the transport on the client
func (a *Client) SetTransport(transport runtime.ClientTransport) {
	a.transport = transport
}
