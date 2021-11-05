package rollrest

import (
	"fmt"
	"github.com/davidji99/simpleresty"
)

// ProjectAccessTokensService handles communication with the project access token related
// methods of the Rollbar API.
//
// Rollbar API docs: https://explorer.docs.rollbar.com/#tag/Project-Access-Tokens
type ProjectAccessTokensService service

// ProjectAccessToken represents a project access token.
type ProjectAccessToken struct {
	ProjectID                   *int64   `json:"project_id,omitempty"`
	AccessToken                 *string  `json:"access_token,omitempty"`
	Name                        *string  `json:"name,omitempty"`
	Status                      *string  `json:"status,omitempty"`
	RateLimitWindowSize         *int     `json:"rate_limit_window_size,omitempty"`
	RateLimitWindowCount        *int     `json:"rate_limit_window_count,omitempty"`
	CurrentRateLimitWindowStart *int64   `json:"cur_rate_limit_window_start,omitempty"`
	CurrentRateLimitWindowCount *int64   `json:"cur_rate_limit_window_count,omitempty"`
	DataCreated                 *int64   `json:"date_created,omitempty"`
	DateModified                *int64   `json:"date_modified,omitempty"`
	Scopes                      []string `json:"scopes,omitempty"`
}

// ProjectAccessTokenResponse represents the response returned after creating a new access token.
type ProjectAccessTokenResponse struct {
	ErrorCount *int                `json:"err,omitempty"`
	Result     *ProjectAccessToken `json:"result,omitempty"`
}

// ProjectAccessTokenListResponse represents the response returned after getting all project access tokens.
type ProjectAccessTokenListResponse struct {
	ErrorCount *int                  `json:"err,omitempty"`
	Result     []*ProjectAccessToken `json:"result,omitempty"`
}

// PATCreateRequest represents a request to create a project access token.
type PATCreateRequest struct {
	Name                 string   `json:"name,omitempty"`
	Scopes               []string `json:"scopes,omitempty"`
	Status               string   `json:"status,omitempty"`
	RateLimitWindowSize  int      `json:"rate_limit_window_size,omitempty"`
	RateLimitWindowCount int      `json:"rate_limit_window_count,omitempty"`
}

// PATUpdateRequest represents a request to update a project access token.
//
// Both RateLimitWindowSize and RateLimitWindowCount need to be set.
type PATUpdateRequest struct {
	RateLimitWindowSize  int `json:"rate_limit_window_size,omitempty"`
	RateLimitWindowCount int `json:"rate_limit_window_count,omitempty"`
}

// List all of a project's access tokens.
//
// Rollbar API docs: https://explorer.docs.rollbar.com/#operation/list-all-project-access-tokens
func (p *ProjectAccessTokensService) List(projectID int) (*ProjectAccessTokenListResponse, *simpleresty.Response, error) {
	var result *ProjectAccessTokenListResponse
	urlStr := p.client.http.RequestURL("/project/%d/access_tokens", projectID)

	// Set the correct authentication header
	p.client.setAuthTokenHeader(p.client.accountAccessToken)

	// Execute the request
	response, getErr := p.client.http.Get(urlStr, &result, nil)

	return result, response, getErr
}

// Get a single project access tokens using the date_created value.
//
// We don't want to use the actual access token.
//
// Also since no endpoint officially exists, this method will first fetch all of a project's access token
// and iterate through each token to find the specified one.
func (p *ProjectAccessTokensService) Get(projectID int, accessToken string) (*ProjectAccessToken, *simpleresty.Response, error) {
	projects, response, listErr := p.List(projectID)
	if listErr != nil {
		return nil, nil, listErr
	}

	var targetProject *ProjectAccessToken

	for _, project := range projects.Result {
		if project.GetAccessToken() == accessToken {
			targetProject = project
		}
	}

	if targetProject == nil {
		return nil, response, fmt.Errorf("specified project access token not found")
	}

	return targetProject, response, nil
}

// Create a project access token.
//
// Rollbar API docs: https://explorer.docs.rollbar.com/#operation/create-a-project-access-token
func (p *ProjectAccessTokensService) Create(projectID int, opts *PATCreateRequest) (*ProjectAccessTokenResponse, *simpleresty.Response, error) {
	var result *ProjectAccessTokenResponse
	urlStr := p.client.http.RequestURL("/project/%d/access_tokens", projectID)

	// Set the correct authentication header
	p.client.setAuthTokenHeader(p.client.accountAccessToken)

	// Execute the request
	response, getErr := p.client.http.Post(urlStr, &result, opts)

	return result, response, getErr
}

// Update a project access token.
//
// Rollbar API docs: https://explorer.docs.rollbar.com/#operation/update-a-rate-limit
func (p *ProjectAccessTokensService) Update(projectID int, accessToken string,
	opts *PATUpdateRequest) (*ProjectAccessTokenResponse, *simpleresty.Response, error) {
	var result *ProjectAccessTokenResponse
	urlStr := p.client.http.RequestURL("/project/%d/access_token/%s", projectID, accessToken)

	// Set the correct authentication header
	p.client.setAuthTokenHeader(p.client.accountAccessToken)

	// Execute the request
	response, getErr := p.client.http.Patch(urlStr, &result, opts)

	return result, response, getErr
}

// TODO: add support for deleting project access tokens when the DELETE endpoint is available.
