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
	"strings"
	"unicode"

	"github.com/pkg/errors"
)

// Client is a client for interacting with a GraphQL API.
type Client struct {
	endpoint         string
	httpClient       *http.Client
	useMultipartForm bool

	// Log is called with various debug information.
	// To log to standard out, use:
	//  client.Log = func(s string) { log.Println(s) }
	Log func(s string)
}

// NewClient makes a new Client capable of making GraphQL requests.
func NewClient(endpoint string, opts ...ClientOption) *Client {
	c := &Client{
		endpoint: endpoint,
		Log:      func(string) {},
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

// Run executes the query and unmarshals the response from the data field
// into the response object.
// Pass in a nil response object to skip response parsing.
// If the request fails or the server returns an error, the first error
// will be returned.
func (c *Client) Run(ctx context.Context, req *Request, resp interface{}) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	if len(req.files) > 0 && !c.useMultipartForm {
		return errors.New("cannot send files with PostFields option")
	}
	if c.useMultipartForm {
		return c.runWithPostFields(ctx, req, resp)
	}
	return c.runWithJSON(ctx, req, resp)
}

func (c *Client) runWithJSON(ctx context.Context, req *Request, resp interface{}) error {
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
	r, err := http.NewRequest(http.MethodPost, c.endpoint, &requestBody)
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

	if req.dataPath != "" {
		val, err := findValueByPath(intermediateResp, req.dataPath)
		if err != nil {
			// If the response is empty, return nil instead of an error
			if len(intermediateResp) == 0 {
				return nil
			}
			return err
		}
		data, err := json.Marshal(val)
		if err != nil {
			return errors.Wrap(err, "remarshaling response")
		}
		return json.Unmarshal(data, resp)
	}

	data, err := json.Marshal(intermediateResp)
	if err != nil {
		return errors.Wrap(err, "remarshaling response")
	}
	if resp == nil {
		return nil
	}
	return json.Unmarshal(data, resp)
}

func (c *Client) runWithPostFields(ctx context.Context, req *Request, resp interface{}) error {
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
	c.logf(">> files: %d", len(req.files))
	c.logf(">> query: %s", req.q)
	intermediateResp := make(map[string]interface{})
	gr := &graphResponse{
		Data: &intermediateResp,
	}
	r, err := http.NewRequest(http.MethodPost, c.endpoint, &requestBody)
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

	if req.dataPath != "" {
		val, err := findValueByPath(intermediateResp, req.dataPath)
		if err != nil {
			return errors.Wrap(err, "finding value by path")
		}
		data, err := json.Marshal(val)
		if err != nil {
			return errors.Wrap(err, "remarshaling response")
		}
		return json.Unmarshal(data, resp)
	}

	data, err := json.Marshal(intermediateResp)
	if err != nil {
		return errors.Wrap(err, "remarshaling response")
	}
	if resp == nil {
		return nil
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
type Request struct {
	q        string
	vars     map[string]interface{}
	files    []file
	dataPath string

	// Header represent any request headers that will be set
	// when the request is made.
	Header http.Header
}

// NewRequest makes a new Request with the specified string.
func NewRequest(q string) *Request {
	req := &Request{
		q:        q,
		Header:   make(map[string][]string),
		dataPath: inferDataPath(q),
	}
	return req
}

// inferDataPath attempts to extract the first field name after the operation type
// as the data path. Returns empty string if unable to infer.
// For example, given the query:
//
//	query { user { name } }
//
// it will return "user".
// The dataPath is used to signal to the client where it should start unmarshaling the response.
func inferDataPath(query string) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return ""
	}

	startIdx := strings.Index(query, "{")
	if startIdx == -1 {
		return ""
	}
	query = query[startIdx+1:]
	query = strings.TrimSpace(query)
	if query == "" || query == "}" {
		return ""
	}

	var result strings.Builder
	for _, ch := range query {
		if ch == '(' || ch == '{' || unicode.IsSpace(ch) || ch == ':' {
			break
		}
		result.WriteRune(ch)
	}

	return strings.TrimSpace(result.String())
}

// Var sets a variable.
func (req *Request) Var(key string, value interface{}) {
	if req.vars == nil {
		req.vars = make(map[string]interface{})
	}
	req.vars[key] = value
}

// File sets a file to upload.
// Files are only supported with a Client that was created with
// the UseMultipartForm option.
func (req *Request) File(fieldname, filename string, r io.Reader) {
	req.files = append(req.files, file{
		Field: fieldname,
		Name:  filename,
		R:     r,
	})
}

// DataPath sets the path to the data field in the response.
// This is useful if you want to unmarshal a nested object.
// If not set, it will use the automatically inferred path.
func (req *Request) DataPath(path string) {
	req.dataPath = path
}

// file represents a file to upload.
type file struct {
	Field string
	Name  string
	R     io.Reader
}

func findValueByPath(data map[string]interface{}, path string) (interface{}, error) {
	if val, ok := data[path]; ok {
		return val, nil
	}

	// Recursively search through nested maps
	for _, val := range data {
		if nestedMap, ok := val.(map[string]interface{}); ok {
			if found, err := findValueByPath(nestedMap, path); err == nil {
				return found, nil
			}
		}
	}

	return nil, fmt.Errorf("path %q not found in response", path)
}
