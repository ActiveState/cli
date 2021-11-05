package rollrest

import "github.com/davidji99/simpleresty"

// UsersService handles communication with the users related
// methods of the Rollbar API.
//
// Rollbar API docs: https://explorer.docs.rollbar.com/#tag/Users
type UsersService service

// User represents a user in Rollbar.
type User struct {
	ID           *int64  `json:"id,omitempty"`
	Username     *string `json:"username,omitempty"`
	Email        *string `json:"email,omitempty"`
	EmailEnabled *int    `json:"email_enabled,omitempty"`
}

// UserResponse represents the response returned after getting a user.
type UserResponse struct {
	ErrorCount *int  `json:"err,omitempty"`
	Result     *User `json:"result,omitempty"`
}

// UserListResponse represents the response returned after getting all users.
type UserListResponse struct {
	ErrorCount *int            `json:"err,omitempty"`
	Result     *UserListResult `json:"result,omitempty"`
}

// UserListResult represents a slice of all users.
type UserListResult struct {
	Users []*User `json:"users,omitempty"`
}

// UserTeamsListResponse represents the response returned from getting a user's teams.
type UserTeamsListResponse struct {
	ErrorCount *int           `json:"err,omitempty"`
	Result     *UserTeamsList `json:"result,omitempty"`
}

// UserTeamsList represents all teams a user belongs to.
type UserTeamsList struct {
	Teams []*Team `json:"teams,omitempty"`
}

// UserProjectsListResponse represents the response returned from getting a user's projects.
type UserProjectsListResponse struct {
	ErrorCount *int              `json:"err,omitempty"`
	Result     *UserProjectsList `json:"result,omitempty"`
}

// UserProjectsList represents all of a user's projects.
type UserProjectsList struct {
	Projects []*UserProject `json:"projects,omitempty"`
}

// UserProject represent's a user's project.
type UserProject struct {
	Status    *int    `json:"status,omitempty"`
	Slug      *string `json:"slug,omitempty"`
	ID        *int64  `json:"id,omitempty"`
	AccountID *int64  `json:"account_id,omitempty"`
}

// List all users.
//
// Rollbar API docs: https://explorer.docs.rollbar.com/#operation/list-all-users
func (u *UsersService) List() (*UserListResponse, *simpleresty.Response, error) {
	var result *UserListResponse
	urlStr := u.client.http.RequestURL("/users")

	// Set the correct authentication header
	u.client.setAuthTokenHeader(u.client.accountAccessToken)

	// Execute the request
	response, getErr := u.client.http.Get(urlStr, &result, nil)

	return result, response, getErr
}

// Get a users.
//
// Returns basic information about the user, as relevant to the account your access token is for.
// This is the same information available on the "Members" page in the Rollbar UI.
//
// Rollbar API docs: https://explorer.docs.rollbar.com/#operation/get-a-user
func (u *UsersService) Get(userID int) (*UserResponse, *simpleresty.Response, error) {
	var result *UserResponse
	urlStr := u.client.http.RequestURL("/user/%d", userID)

	// Set the correct authentication header
	u.client.setAuthTokenHeader(u.client.accountAccessToken)

	// Execute the request
	response, getErr := u.client.http.Get(urlStr, &result, nil)

	return result, response, getErr
}

// ListTeams lists all teams that a user is a member of.
//
// Rollbar API docs: https://explorer.docs.rollbar.com/#operation/list-a-users-teams
func (u *UsersService) ListTeams(userID int) (*UserTeamsListResponse, *simpleresty.Response, error) {
	var result *UserTeamsListResponse
	urlStr := u.client.http.RequestURL("/user/%d/teams", userID)

	// Set the correct authentication header
	u.client.setAuthTokenHeader(u.client.accountAccessToken)

	// Execute the request
	response, getErr := u.client.http.Get(urlStr, &result, nil)

	return result, response, getErr
}

// ListProjects lists all of a user's projects.
//
// Rollbar API docs: https://explorer.docs.rollbar.com/#operation/list-a-users-projects
func (u *UsersService) ListProjects(userID int) (*UserProjectsListResponse, *simpleresty.Response, error) {
	var result *UserProjectsListResponse
	urlStr := u.client.http.RequestURL("/user/%d/projects", userID)

	// Set the correct authentication header
	u.client.setAuthTokenHeader(u.client.accountAccessToken)

	// Execute the request
	response, getErr := u.client.http.Get(urlStr, &result, nil)

	return result, response, getErr
}
