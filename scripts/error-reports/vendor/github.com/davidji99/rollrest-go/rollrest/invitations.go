package rollrest

import (
	"github.com/davidji99/simpleresty"
)

// InvitationsService handles communication with the invitation related
// methods of the Rollbar API.
//
// Rollbar API docs: N/A
type InvitationsService service

// InvitationResponse represents a response after inviting an user.
type InvitationResponse struct {
	ErrorCount *int        `json:"err,omitempty"`
	Result     *Invitation `json:"result,omitempty"`
	Message    *string     `json:"message,omitempty"`
}

// InvitationListResponse represents a response of all invitations.
type InvitationListResponse struct {
	ErrorCount *int          `json:"err,omitempty"`
	Result     []*Invitation `json:"result,omitempty"`
}

// Invitation represents an invitation in Rollbar (usually an user's invitation to a team).
type Invitation struct {
	ID           *int64  `json:"id,omitempty"`
	FromUserID   *int64  `json:"from_user_id,omitempty"`
	TeamID       *int64  `json:"team_id,omitempty"`
	ToEmail      *string `json:"to_email,omitempty"`
	Status       *string `json:"status,omitempty"`
	DateCreated  *int64  `json:"date_created,omitempty"`
	DateRedeemed *int64  `json:"date_redeemed,omitempty"`
}

// Get an invitation.
//
// Rollbar API docs: https://explorer.docs.rollbar.com/#operation/get-invitation
func (i *InvitationsService) Get(inviteID int) (*InvitationResponse, *simpleresty.Response, error) {
	var result *InvitationResponse
	urlStr := i.client.http.RequestURL("/invite/%d", inviteID)

	// Set the correct authentication header
	i.client.setAuthTokenHeader(i.client.accountAccessToken)

	// Execute the request
	response, getErr := i.client.http.Get(urlStr, &result, nil)

	return result, response, getErr
}

// Cancel an invitation.
//
// Rollbar API docs: https://explorer.docs.rollbar.com/#operation/cancel-invitation
func (i *InvitationsService) Cancel(inviteID int) (*GenericResponse, *simpleresty.Response, error) {
	var result GenericResponse

	urlStr := i.client.http.RequestURL("/invite/%d", inviteID)

	// Set the correct authentication header
	i.client.setAuthTokenHeader(i.client.accountAccessToken)

	// Execute the request
	response, getErr := i.client.http.Delete(urlStr, &result, nil)

	return &result, response, getErr
}
