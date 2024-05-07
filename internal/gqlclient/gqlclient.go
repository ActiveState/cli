package gqlclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/profile"
	"github.com/ActiveState/cli/internal/singleton/uniqid"
	"github.com/ActiveState/cli/internal/strutils"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/graphql"
	"github.com/pkg/errors"
)

type File struct {
	Field string
	Name  string
	R     io.Reader
}

type Request0 interface {
	Query() string
	Vars() map[string]interface{}
}

type Request interface {
	Query() string
	Vars() (map[string]interface{}, error)
}

type RequestWithFiles interface {
	Request
	Files() []File
}

type Header map[string][]string

type graphqlClient = graphql.Client

// StandardizedErrors works around API's that don't follow the graphql standard
// It looks redundant because it needs to address two different API responses.
// https://activestatef.atlassian.net/browse/PB-4291
type StandardizedErrors struct {
	Message string
	Error   string
	Errors  []graphErr
}

func (e StandardizedErrors) HasErrors() bool {
	return len(e.Errors) > 0 || e.Error != ""
}

// Values tells us all the relevant error messages returned.
// We don't include e.Error because it's an unhelpful generic error code redundant with the message.
func (e StandardizedErrors) Values() []string {
	var errs []string
	for _, err := range e.Errors {
		errs = append(errs, err.Message)
	}
	if e.Message != "" {
		errs = append(errs, e.Message)
	}
	return errs
}

type graphResponse struct {
	Data    interface{}
	Error   string
	Message string
	Errors  []graphErr
}

type graphErr struct {
	Message string
}

func (e graphErr) Error() string {
	return "graphql: " + e.Message
}

type BearerTokenProvider interface {
	BearerToken() string
}

type Client struct {
	*graphqlClient
	url           string
	tokenProvider BearerTokenProvider
	timeout       time.Duration
}

func NewWithOpts(url string, timeout time.Duration, opts ...graphql.ClientOption) *Client {
	if timeout == 0 {
		timeout = time.Second * 60
	}

	client := &Client{
		graphqlClient: graphql.NewClient(url, opts...),
		timeout:       timeout,
		url:           url,
	}
	if os.Getenv(constants.DebugServiceRequestsEnvVarName) == "true" {
		client.EnableDebugLog()
	}
	return client
}

func New(url string, timeout time.Duration) *Client {
	return NewWithOpts(url, timeout, graphql.WithHTTPClient(api.NewHTTPClient()))
}

// EnableDebugLog turns on debug logging
func (c *Client) EnableDebugLog() {
	c.graphqlClient.Log = func(s string) { logging.Debug("graphqlClient log message: %s", s) }
}

func (c *Client) SetTokenProvider(tokenProvider BearerTokenProvider) {
	c.tokenProvider = tokenProvider
}

func (c *Client) SetDebug(b bool) {
	c.graphqlClient.Log = func(string) {}
	if b {
		c.graphqlClient.Log = func(s string) {
			fmt.Fprintln(os.Stderr, s)
		}
	}
}

func (c *Client) Run(request Request, response interface{}) error {
	ctx := context.Background()
	if c.timeout != 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}
	err := c.RunWithContext(ctx, request, response)
	return err // Needs var so the cancel defer triggers at the right time
}

type PostProcessor interface {
	PostProcess() error
}

func (c *Client) RunWithContext(ctx context.Context, request Request, response interface{}) (rerr error) {
	defer func() {
		if rerr != nil {
			return
		}
		if postProcessor, ok := response.(PostProcessor); ok {
			rerr = postProcessor.PostProcess()
		}
	}()
	name := strutils.Summarize(request.Query(), 25)
	defer profile.Measure(fmt.Sprintf("gqlclient:RunWithContext:(%s)", name), time.Now())

	if fileRequest, ok := request.(RequestWithFiles); ok {
		return c.runWithFiles(ctx, fileRequest, response)
	}

	vars, err := request.Vars()
	if err != nil {
		return errs.Wrap(err, "Could not get variables")
	}

	graphRequest := graphql.NewRequest(request.Query())
	for key, value := range vars {
		graphRequest.Var(key, value)
	}

	if fileRequest, ok := request.(RequestWithFiles); ok {
		for _, file := range fileRequest.Files() {
			graphRequest.File(file.Field, file.Name, file.R)
		}
	}

	var bearerToken string
	if c.tokenProvider != nil {
		bearerToken = c.tokenProvider.BearerToken()
		if bearerToken != "" {
			graphRequest.Header.Set("Authorization", "Bearer "+bearerToken)
		}
	}

	graphRequest.Header.Set("X-Requestor", uniqid.Text())

	if err := c.graphqlClient.Run(ctx, graphRequest, &response); err != nil {
		return NewRequestError(err, request)
	}

	return nil
}

type JsonRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

func (c *Client) runWithFiles(ctx context.Context, gqlReq RequestWithFiles, response interface{}) error {
	// Construct the multi-part request.
	bodyReader, bodyWriter := io.Pipe()

	req, err := http.NewRequest("POST", c.url, bodyReader)
	if err != nil {
		return errs.Wrap(err, "Could not create http request")
	}

	req.Body = bodyReader

	mw := multipart.NewWriter(bodyWriter)
	req.Header.Set("Content-Type", "multipart/form-data; boundary="+mw.Boundary())

	vars, err := gqlReq.Vars()
	if err != nil {
		return errs.Wrap(err, "Could not get variables")
	}

	varJson, err := json.Marshal(vars)
	if err != nil {
		return errs.Wrap(err, "Could not marshal vars")
	}

	reqErrChan := make(chan error)
	go func() {
		defer bodyWriter.Close()
		defer mw.Close()
		defer close(reqErrChan)

		// Operations
		operations, err := mw.CreateFormField("operations")
		if err != nil {
			reqErrChan <- errs.Wrap(err, "Could not create form field operations")
			return
		}

		jsonReq := JsonRequest{
			Query:     gqlReq.Query(),
			Variables: vars,
		}
		jsonReqV, err := json.Marshal(jsonReq)
		if err != nil {
			reqErrChan <- errs.Wrap(err, "Could not marshal json request")
			return
		}
		if _, err := operations.Write(jsonReqV); err != nil {
			reqErrChan <- errs.Wrap(err, "Could not write json request")
			return
		}

		// Map
		if len(gqlReq.Files()) > 0 {
			mapField, err := mw.CreateFormField("map")
			if err != nil {
				reqErrChan <- errs.Wrap(err, "Could not create form field map")
				return
			}
			for n, f := range gqlReq.Files() {
				if _, err := mapField.Write([]byte(fmt.Sprintf(`{"%d": ["%s"]}`, n, f.Field))); err != nil {
					reqErrChan <- errs.Wrap(err, "Could not write map field")
					return
				}
			}
			// File upload
			for n, file := range gqlReq.Files() {
				part, err := mw.CreateFormFile(fmt.Sprintf("%d", n), file.Name)
				if err != nil {
					reqErrChan <- errs.Wrap(err, "Could not create form file")
					return
				}

				_, err = io.Copy(part, file.R)
				if err != nil {
					reqErrChan <- errs.Wrap(err, "Could not read file")
					return
				}
			}
		}
	}()

	c.Log(fmt.Sprintf(">> query: %s", gqlReq.Query()))
	c.Log(fmt.Sprintf(">> variables: %s", string(varJson)))
	fnames := []string{}
	for _, file := range gqlReq.Files() {
		fnames = append(fnames, file.Name)
	}
	c.Log(fmt.Sprintf(">> files: %v", fnames))

	// Run the request.
	var bearerToken string
	if c.tokenProvider != nil {
		bearerToken = c.tokenProvider.BearerToken()
		if bearerToken != "" {
			req.Header.Set("Authorization", "Bearer "+bearerToken)
		}
	}
	if os.Getenv(constants.DebugServiceRequestsEnvVarName) == "true" {
		responseData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return errs.Wrap(err, "failed to marshal response")
		}
		logging.Debug("gqlclient: response: %s", responseData)
	}

	gr := &graphResponse{
		Data: response,
	}
	req = req.WithContext(ctx)
	c.Log(fmt.Sprintf(">> Raw Request: %s\n", req.URL.String()))

	var res *http.Response
	resErrChan := make(chan error)
	go func() {
		var err error
		res, err = http.DefaultClient.Do(req)
		resErrChan <- err
	}()

	// Due to the streaming uploads the request error can happen both before and after the http request itself, hence
	// the creative select case you see before you.
	wait := true
	for wait {
		select {
		case err := <-reqErrChan:
			if err != nil {
				c.Log(fmt.Sprintf("Request Error: %s", err))
				return err
			}
		case err := <-resErrChan:
			wait = false
			if err != nil {
				c.Log(fmt.Sprintf("Response Error: %s", err))
				return err
			}
		}
	}

	if res == nil {
		return errs.New("Received empty response")
	}

	defer res.Body.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, res.Body); err != nil {
		c.Log(fmt.Sprintf("Read Error: %s", err))
		return errors.Wrap(err, "reading body")
	}
	resp := buf.Bytes()
	c.Log(fmt.Sprintf("<< Response code: %d, body: %s\n", res.StatusCode, string(resp)))

	// Work around API's that don't follow the graphql standard
	// https://activestatef.atlassian.net/browse/PB-4291
	standardizedErrors := StandardizedErrors{}
	if err := json.Unmarshal(resp, &standardizedErrors); err != nil {
		return errors.Wrap(err, "decoding error response")
	}
	if standardizedErrors.HasErrors() {
		return errs.New(strings.Join(standardizedErrors.Values(), "\n"))
	}

	if err := json.Unmarshal(resp, &gr); err != nil {
		return errors.Wrap(err, "decoding response")
	}
	return nil
}
