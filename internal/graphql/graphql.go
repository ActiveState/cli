// Package graphql provides a low level GraphQL client.
//
//	// create a client (safe to share across requests)
//	client := graphql.NewClient("https://machinebox.io/graphql")
//
//	// make a request
//	req := graphql.NewRequest(`
//	    query ($key: String!) {
//	        items (id:$key) {
//	            field1
//	            field2
//	            field3
//	        }
//	    }
//	`)
//
//	// set any variables
//	req.Var("key", "value")
//
//	// run it and capture the response
//	var respData ResponseStruct
//	if err := client.Run(ctx, req, &respData); err != nil {
//	    log.Fatal(err)
//	}
//
// # Specify client
//
// To specify your own http.Client, use the WithHTTPClient option:
//
//	httpclient := &http.Client{}
//	client := graphql.NewClient("https://machinebox.io/graphql", graphql.WithHTTPClient(httpclient))
package graphql

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
	"github.com/pkg/errors"
)

type Request interface {
	Query() string
	Vars() (map[string]interface{}, error)
}

type RequestWithFiles interface {
	Request
	Files() []File
}

type RequestWithHeaders interface {
	Request
	Headers() map[string][]string
}

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

type graphErr struct {
	Message string
}

func (e graphErr) Error() string {
	return "graphql: " + e.Message
}

type BearerTokenProvider interface {
	BearerToken() string
}

type PostProcessor interface {
	PostProcess() error
}

// Client is a client for interacting with a GraphQL API.
type Client struct {
	url              string
	httpClient       *http.Client
	useMultipartForm bool
	tokenProvider    BearerTokenProvider
	timeout          time.Duration

	// Log is called with various debug information.
	// To log to standard out, use:
	//  client.Log = func(s string) { log.Println(s) }
	Log func(s string)
}

func NewWithOpts(url string, timeout time.Duration, opts ...ClientOption) *Client {
	if timeout == 0 {
		timeout = time.Second * 60
	}

	c := newClient(url, opts...)
	if os.Getenv(constants.DebugServiceRequestsEnvVarName) == "true" {
		c.EnableDebugLog()
	}
	c.timeout = timeout

	return c
}

func New(url string, timeout time.Duration) *Client {
	return NewWithOpts(url, timeout, WithHTTPClient(api.NewHTTPClient()))
}

// newClient makes a new Client capable of making GraphQL requests.
func newClient(endpoint string, opts ...ClientOption) *Client {
	c := &Client{
		url: endpoint,
		Log: func(string) {},
	}
	for _, optionFunc := range opts {
		optionFunc(c)
	}
	if c.httpClient == nil {
		c.httpClient = http.DefaultClient
	}
	return c
}

func (c *Client) logf(format string, args ...interface{}) {
	c.Log(fmt.Sprintf(format, args...))
}

func (c *Client) EnableDebugLog() {
	c.Log = func(s string) {
		logging.Debug("graphqlClient log message: %s", s)
	}
}

func (c *Client) SetTokenProvider(tokenProvider BearerTokenProvider) {
	c.tokenProvider = tokenProvider
}

func (c *Client) Run(request Request, response interface{}) error {
	ctx := context.Background()
	if c.timeout != 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}
	if err := c.RunWithContext(ctx, request, response); err != nil {
		return NewRequestError(err, request)
	}
	return nil
}

// RunWithContext executes the query and unmarshals the response from the data field
// into the response object.
// Pass in a nil response object to skip response parsing.
// If the request fails or the server returns an error, the first error
// will be returned.
func (c *Client) RunWithContext(ctx context.Context, req Request, resp interface{}) (rerr error) {
	defer func() {
		if rerr != nil {
			return
		}
		if postProcessor, ok := resp.(PostProcessor); ok {
			rerr = postProcessor.PostProcess()
		}
	}()
	name := strutils.Summarize(req.Query(), 25)
	defer profile.Measure(fmt.Sprintf("gqlclient:RunWithContext:(%s)", name), time.Now())

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	gqlRequest := newRequest(req.Query())
	vars, err := req.Vars()
	if err != nil {
		return errs.Wrap(err, "Could not get vars")
	}
	gqlRequest.vars = vars

	var bearerToken string
	if c.tokenProvider != nil {
		bearerToken = c.tokenProvider.BearerToken()
		if bearerToken != "" {
			gqlRequest.Header.Set("Authorization", "Bearer "+bearerToken)
		}
	}

	gqlRequest.Header.Set("X-Requestor", uniqid.Text())

	if header, ok := req.(RequestWithHeaders); ok {
		for key, values := range header.Headers() {
			for _, value := range values {
				gqlRequest.Header.Add(key, value)
			}
		}
	}

	if fileRequest, ok := req.(RequestWithFiles); ok {
		gqlRequest.files = fileRequest.Files()
		return c.runWithFiles(ctx, gqlRequest, resp)
	}

	if c.useMultipartForm {
		return c.runWithPostFields(ctx, gqlRequest, resp)
	}

	return c.runWithJSON(ctx, gqlRequest, resp)
}

func (c *Client) runWithJSON(ctx context.Context, req *gqlRequest, resp interface{}) error {
	var requestBody bytes.Buffer
	requestBodyObj := struct {
		Query     string                 `json:"query"`
		Variables map[string]interface{} `json:"variables"`
	}{
		Query:     req.q,
		Variables: req.vars,
	}
	if err := json.NewEncoder(&requestBody).Encode(requestBodyObj); err != nil {
		return errors.Wrap(err, "encode body")
	}
	c.logf(">> variables: %v", req.vars)
	c.logf(">> query: %s", req.q)

	intermediateResp := make(map[string]interface{})
	gr := &graphResponse{
		Data: &intermediateResp,
	}
	r, err := http.NewRequest(http.MethodPost, c.url, &requestBody)
	if err != nil {
		return err
	}
	r.Header.Set("Content-Type", "application/json; charset=utf-8")
	r.Header.Set("Accept", "application/json; charset=utf-8")
	for key, values := range req.Header {
		for _, value := range values {
			r.Header.Add(key, value)
		}
	}
	c.logf(">> headers: %v", r.Header)
	r = r.WithContext(ctx)
	res, err := c.httpClient.Do(r)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, res.Body); err != nil {
		return errors.Wrap(err, "reading body")
	}
	c.logf("<< %s", buf.String())
	if err := json.NewDecoder(&buf).Decode(&gr); err != nil {
		return errors.Wrap(err, "decoding response")
	}
	if len(gr.Errors) > 0 {
		// return first error
		return gr.Errors[0]
	}

	return c.marshalResponse(intermediateResp, resp)
}

func (c *Client) runWithPostFields(ctx context.Context, req *gqlRequest, resp interface{}) error {
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)
	if err := writer.WriteField("query", req.q); err != nil {
		return errors.Wrap(err, "write query field")
	}
	var variablesBuf bytes.Buffer
	if len(req.vars) > 0 {
		variablesField, err := writer.CreateFormField("variables")
		if err != nil {
			return errors.Wrap(err, "create variables field")
		}
		if err := json.NewEncoder(io.MultiWriter(variablesField, &variablesBuf)).Encode(req.vars); err != nil {
			return errors.Wrap(err, "encode variables")
		}
	}

	for i := range req.files {
		part, err := writer.CreateFormFile(req.files[i].Field, req.files[i].Name)
		if err != nil {
			return errors.Wrap(err, "create form file")
		}
		if _, err := io.Copy(part, req.files[i].R); err != nil {
			return errors.Wrap(err, "preparing file")
		}
	}

	if err := writer.Close(); err != nil {
		return errors.Wrap(err, "close writer")
	}

	c.logf(">> variables: %s", variablesBuf.String())
	c.logf(">> query: %s", req.q)
	c.logf(">> files: %d", len(req.files))
	intermediateResp := make(map[string]interface{})
	gr := &graphResponse{
		Data: &intermediateResp,
	}
	r, err := http.NewRequest(http.MethodPost, c.url, &requestBody)
	if err != nil {
		return err
	}
	r.Header.Set("Content-Type", writer.FormDataContentType())
	r.Header.Set("Accept", "application/json; charset=utf-8")
	for key, values := range req.Header {
		for _, value := range values {
			r.Header.Add(key, value)
		}
	}
	c.logf(">> headers: %v", r.Header)
	r = r.WithContext(ctx)
	res, err := c.httpClient.Do(r)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, res.Body); err != nil {
		return errors.Wrap(err, "reading body")
	}
	c.logf("<< %s", buf.String())
	if err := json.NewDecoder(&buf).Decode(&gr); err != nil {
		return errors.Wrap(err, "decoding response")
	}
	if len(gr.Errors) > 0 {
		// return first error
		return gr.Errors[0]
	}

	return c.marshalResponse(intermediateResp, resp)
}

type jsonRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

func (c *Client) runWithFiles(ctx context.Context, request *gqlRequest, response interface{}) error {
	// Construct the multi-part request.
	bodyReader, bodyWriter := io.Pipe()

	req, err := http.NewRequest("POST", c.url, bodyReader)
	if err != nil {
		return errs.Wrap(err, "Could not create http request")
	}

	req.Body = bodyReader

	mw := multipart.NewWriter(bodyWriter)
	req.Header.Set("Content-Type", "multipart/form-data; boundary="+mw.Boundary())

	vars := request.vars
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

		jsonReq := jsonRequest{
			Query:     request.q,
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
		if len(request.files) > 0 {
			mapField, err := mw.CreateFormField("map")
			if err != nil {
				reqErrChan <- errs.Wrap(err, "Could not create form field map")
				return
			}
			for n, f := range request.files {
				if _, err := mapField.Write([]byte(fmt.Sprintf(`{"%d": ["%s"]}`, n, f.Field))); err != nil {
					reqErrChan <- errs.Wrap(err, "Could not write map field")
					return
				}
			}
			// File upload
			for n, file := range request.files {
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

	c.Log(fmt.Sprintf(">> query: %s", request.q))
	c.Log(fmt.Sprintf(">> variables: %s", string(varJson)))
	fnames := []string{}
	for _, file := range request.files {
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

	intermediateResp := make(map[string]interface{})
	gr := &graphResponse{
		Data: &intermediateResp,
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

	// If the response is a single object, meaning we only have a single query in the request, we can unmarshal the
	// response directly to the response type. Otherwise, we need to marshal the response as we normally would.
	if len(intermediateResp) == 1 {
		for _, val := range intermediateResp {
			data, err := json.Marshal(val)
			if err != nil {
				return errors.Wrap(err, "remarshaling response")
			}
			return json.Unmarshal(data, response)
		}
	}

	data, err := json.Marshal(intermediateResp)
	if err != nil {
		return errors.Wrap(err, "remarshaling response")
	}
	return json.Unmarshal(data, response)
}

func (c *Client) marshalResponse(intermediateResp map[string]interface{}, resp interface{}) error {
	if resp == nil {
		return nil
	}

	if len(intermediateResp) == 1 {
		for _, val := range intermediateResp {
			data, err := json.Marshal(val)
			if err != nil {
				return errors.Wrap(err, "remarshaling response")
			}
			return json.Unmarshal(data, resp)
		}
	}

	data, err := json.Marshal(intermediateResp)
	if err != nil {
		return errors.Wrap(err, "remarshaling response")
	}
	return json.Unmarshal(data, resp)
}

// WithHTTPClient specifies the underlying http.Client to use when
// making requests.
//
//	NewClient(endpoint, WithHTTPClient(specificHTTPClient))
func WithHTTPClient(httpclient *http.Client) ClientOption {
	return func(client *Client) {
		client.httpClient = httpclient
	}
}

// UseMultipartForm uses multipart/form-data and activates support for
// files.
func UseMultipartForm() ClientOption {
	return func(client *Client) {
		client.useMultipartForm = true
	}
}

// ClientOption are functions that are passed into NewClient to
// modify the behaviour of the Client.
type ClientOption func(*Client)

type GraphErr struct {
	Message    string                 `json:"message"`
	Extensions map[string]interface{} `json:"extensions"`
}

func (e GraphErr) Error() string {
	return "graphql: " + e.Message
}

type graphResponse struct {
	Data   interface{}
	Errors []GraphErr
}

// Request is a GraphQL request.
type gqlRequest struct {
	q     string
	vars  map[string]interface{}
	files []File

	// Header represent any request headers that will be set
	// when the request is made.
	Header http.Header
}

// newRequest makes a new Request with the specified string.
func newRequest(q string) *gqlRequest {
	req := &gqlRequest{
		q:      q,
		Header: make(map[string][]string),
	}
	return req
}

// Var sets a variable.
func (req *gqlRequest) Var(key string, value interface{}) {
	if req.vars == nil {
		req.vars = make(map[string]interface{})
	}
	req.vars[key] = value
}

// File sets a file to upload.
// Files are only supported with a Client that was created with
// the UseMultipartForm option.
func (req *gqlRequest) File(fieldname, filename string, r io.Reader) {
	req.files = append(req.files, File{
		Field: fieldname,
		Name:  filename,
		R:     r,
	})
}

// File represents a File to upload.
type File struct {
	Field string
	Name  string
	R     io.Reader
}
