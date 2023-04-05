package gqlclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/profile"
	"github.com/ActiveState/cli/internal/retryhttp"
	"github.com/ActiveState/cli/internal/singleton/uniqid"
	"github.com/ActiveState/cli/internal/strutils"
	"github.com/machinebox/graphql"
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

type RequestBase struct{}

func (r *RequestBase) Files() []File {
	return []File{}
}

type Request interface {
	Files() []File
	Query() string
	Vars() map[string]interface{}
}

type Header map[string][]string

type graphqlClient = graphql.Client

type graphResponse struct {
	Data   interface{}
	Errors []graphErr
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
	return NewWithOpts(url, timeout, graphql.WithHTTPClient(retryhttp.DefaultClient.StandardClient()))
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

func (c *Client) RunWithContext(ctx context.Context, request Request, response interface{}) error {
	name := strutils.Summarize(request.Query(), 25)
	defer profile.Measure(fmt.Sprintf("gqlclient:RunWithContext:(%s)", name), time.Now())

	if len(request.Files()) > 0 {
		return c.runWithFiles(ctx, request, response)
	}

	graphRequest := graphql.NewRequest(request.Query())
	for key, value := range request.Vars() {
		graphRequest.Var(key, value)
	}

	for _, file := range request.Files() {
		graphRequest.File(file.Field, file.Name, file.R)
	}

	var bearerToken string
	if c.tokenProvider != nil {
		bearerToken = c.tokenProvider.BearerToken()
		if bearerToken != "" {
			graphRequest.Header.Set("Authorization", "Bearer "+bearerToken)
		}
	}

	graphRequest.Header.Set("X-Requestor", uniqid.Text())

	err := c.graphqlClient.Run(ctx, graphRequest, &response)
	if err != nil {
		return NewRequestError(err, request)
	}

	return nil
}

func (c *Client) runRaw(ctx context.Context, request *http.Request, response interface{}) error {
	var bearerToken string
	if c.tokenProvider != nil {
		bearerToken = c.tokenProvider.BearerToken()
		if bearerToken != "" {
			request.Header.Set("Authorization", "Bearer "+bearerToken)
		}
	}

	gr := &graphResponse{
		Data: response,
	}
	request = request.WithContext(ctx)
	res, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, res.Body); err != nil {
		return errors.Wrap(err, "reading body")
	}
	if err := json.NewDecoder(&buf).Decode(&gr); err != nil {
		return errors.Wrap(err, "decoding response")
	}
	if len(gr.Errors) > 0 {
		return gr.Errors[0]
	}
	return nil
}

func (c *Client) runWithFiles(ctx context.Context, gqlReq Request, response interface{}) error {
	req, err := c.createMultiPartUploadRequest(gqlReq)
	if err != nil {
		return errs.Wrap(err, "Could not create multipart upload request")
	}

	return c.runRaw(ctx, req, response)
}

type JsonRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

func (c *Client) createMultiPartUploadRequest(gqlReq Request) (*http.Request, error) {
	req, err := http.NewRequest("POST", c.url, nil)
	if err != nil {
		return nil, errs.Wrap(err, "Could not create http request")
	}

	body := bytes.NewBuffer([]byte{})
	req.Body = ioutil.NopCloser(body)

	mw := multipart.NewWriter(body)
	req.Header.Set("Content-Type", "multipart/form-data; boundary="+mw.Boundary())

	// Operations
	operations, err := mw.CreateFormField("operations")
	if err != nil {
		return nil, errs.Wrap(err, "Could not create form field operations")
	}
	jsonReq := JsonRequest{
		Query:     gqlReq.Query(),
		Variables: gqlReq.Vars(),
	}
	jsonReqV, err := json.Marshal(jsonReq)
	if err != nil {
		return nil, errs.Wrap(err, "Could not marshal json request")
	}
	if _, err := operations.Write(jsonReqV); err != nil {
		return nil, errs.Wrap(err, "Could not write json request")
	}

	// Map
	mapField, err := mw.CreateFormField("map")
	if err != nil {
		return nil, errs.Wrap(err, "Could not create form field map")
	}
	for n := range gqlReq.Files() {
		if _, err := mapField.Write([]byte(fmt.Sprintf(`{"%d": ["variables.file"]}`, n))); err != nil {
			return nil, errs.Wrap(err, "Could not write map field")
		}
	}

	// File upload
	for n, file := range gqlReq.Files() {
		w, err := mw.CreateFormFile(fmt.Sprintf("%d", n), file.Name)
		if err != nil {
			return nil, errs.Wrap(err, "Could not create form file")
		}

		b, err := ioutil.ReadAll(file.R)
		if err != nil {
			return nil, errs.Wrap(err, "Could not read file")
		}

		if _, err := w.Write(b); err != nil {
			return nil, errs.Wrap(err, "Could not write file")
		}
	}

	if err := mw.Close(); err != nil {
		return nil, errs.Wrap(err, "Could not close multipart writer")
	}

	return req, nil
}
