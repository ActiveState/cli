package rollrest

import (
	"fmt"
	"sync"
	"time"

	"github.com/davidji99/simpleresty"
)

const (
	// DefaultAPIBaseURL is the base URL when making API calls.
	DefaultAPIBaseURL = "https://api.rollbar.com/api/1"

	// DefaultUserAgent is the user agent used when making API calls.
	DefaultUserAgent = "rollrest-go"

	// RollbarAuthHeader is the Authorization header.
	RollbarAuthHeader = "x-rollbar-access-token"
)

// A Client manages communication with the Rollbar API.
type Client struct {
	// clientMu protects the client during calls that modify the CheckRedirect func.
	clientMu sync.Mutex

	// HTTP client used to communicate with the API.
	http *simpleresty.Client

	// baseURL for API. No trailing slashes.
	baseURL string

	// Reuse a single struct instead of allocating one for each service on the heap.
	common service

	// User agent used when communicating with the Rollbar API.
	userAgent string

	// Custom HTTPHeaders
	customHTTPHeaders map[string]string

	// Account access token
	accountAccessToken string

	// Project access token
	projectAccessToken string

	// Services used for talking to different parts of the Rollbar API.
	Invitations         *InvitationsService
	Notifications       *NotificationsService
	Projects            *ProjectsService
	ProjectAccessTokens *ProjectAccessTokensService
	Teams               *TeamsService
	Users               *UsersService
	RQL                 *RQLService
}

// service represents the API service client.
type service struct {
	client *Client
}

// GenericResponse represents a generic response from Rollbar.
type GenericResponse struct {
	Err     *int    `json:"err,omitempty"`
	Message *string `json:"message,omitempty"`
}

// New constructs a new client to interact with the API using a project and/or account access token.
func New(opts ...Option) (*Client, error) {
	// Construct new client.
	c := &Client{
		http:               simpleresty.New(),
		baseURL:            DefaultAPIBaseURL,
		userAgent:          DefaultUserAgent,
		customHTTPHeaders:  map[string]string{},
		accountAccessToken: "",
		projectAccessToken: "",
	}

	// Define any user custom Client settings
	if optErr := c.parseOptions(opts...); optErr != nil {
		return nil, optErr
	}

	// Validate that Client has a non empty account or project access token. One must be set.
	if c.accountAccessToken == "" && c.projectAccessToken == "" {
		return nil, fmt.Errorf("please set one or both: acccount/project access token")
	}

	// Setup the client with default settings
	c.setupClient()

	// Inject services
	c.injectServices()

	return c, nil
}

// injectServices adds the services to the client.
func (c *Client) injectServices() {
	c.common.client = c
	c.Invitations = (*InvitationsService)(&c.common)
	c.Notifications = (*NotificationsService)(&c.common)
	c.Projects = (*ProjectsService)(&c.common)
	c.ProjectAccessTokens = (*ProjectAccessTokensService)(&c.common)
	c.Teams = (*TeamsService)(&c.common)
	c.Users = (*UsersService)(&c.common)
	c.RQL = (*RQLService)(&c.common)
}

// setupClient sets common headers and other configurations.
func (c *Client) setupClient() {
	// Set Base URL
	c.http.SetBaseURL(c.baseURL)

	/*
		We aren't setting an authentication header initially here as certain API resources require specific access_tokens.
		Per Rollbar API documentation, each individual resource will set the access_token parameter when constructing
		the full API endpoint URL.
	*/
	c.http.SetHeader("Content-type", "application/json").
		SetHeader("Accept", "application/json").
		SetHeader("User-Agent", c.userAgent).
		SetTimeout(1 * time.Minute).
		SetAllowGetMethodPayload(true)

	// Set additional headers
	if c.customHTTPHeaders != nil {
		c.http.SetHeaders(c.customHTTPHeaders)
	}
}

// parseOptions parses the supplied options functions and returns a configured *Client instance.
func (c *Client) parseOptions(opts ...Option) error {
	// Range over each options function and apply it to our API type to
	// configure it. Options functions are applied in order, with any
	// conflicting options overriding earlier calls.
	for _, option := range opts {
		err := option(c)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) setAuthTokenHeader(token string) {
	c.http.SetHeader(RollbarAuthHeader, token)
}
