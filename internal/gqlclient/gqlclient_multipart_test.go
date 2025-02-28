package gqlclient

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/matryer/is"
)

func TestWithClient(t *testing.T) {
	is := is.New(t)
	var calls int
	testClient := &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			resp := &http.Response{
				Body: ioutil.NopCloser(strings.NewReader(`{"data":{"key":"value"}}`)),
			}
			return resp, nil
		}),
	}

	ctx := context.Background()
	client := newClient("", WithHTTPClient(testClient), UseMultipartForm())

	client.RunWithContext(ctx, NewTestRequest("query {}"), nil)

	is.Equal(calls, 1) // calls
}

func TestDoUseMultipartForm(t *testing.T) {
	is := is.New(t)
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		is.Equal(r.Method, http.MethodPost)
		query := r.FormValue("query")
		is.Equal(query, `query {}`)
		io.WriteString(w, `{
			"data": {
				"query": {
					"something": "yes"
				}
			}
		}`)
	}))
	defer srv.Close()

	ctx := context.Background()
	client := newClient(srv.URL, UseMultipartForm())

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	var responseData map[string]interface{}
	err := client.RunWithContext(ctx, NewTestRequest("query {}"), &responseData)
	is.NoErr(err)
	is.Equal(calls, 1) // calls
	is.Equal(responseData["something"], "yes")
}

func TestDoErr(t *testing.T) {
	is := is.New(t)
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		is.Equal(r.Method, http.MethodPost)
		query := r.FormValue("query")
		is.Equal(query, `query {}`)
		io.WriteString(w, `{
			"errors": [{
				"message": "Something went wrong"
			}]
		}`)
	}))
	defer srv.Close()

	ctx := context.Background()
	client := newClient(srv.URL, UseMultipartForm())

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	var responseData map[string]interface{}
	err := client.RunWithContext(ctx, NewTestRequest("query {}"), &responseData)
	is.True(err != nil)
	is.Equal(err.Error(), "graphql: Something went wrong")
}

func TestDoNoResponse(t *testing.T) {
	is := is.New(t)
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		is.Equal(r.Method, http.MethodPost)
		query := r.FormValue("query")
		is.Equal(query, `query {}`)
		io.WriteString(w, `{
			"data": {
				"something": "yes"
			}
		}`)
	}))
	defer srv.Close()

	ctx := context.Background()
	client := newClient(srv.URL, UseMultipartForm())

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	err := client.RunWithContext(ctx, NewTestRequest("query {}"), nil)
	is.NoErr(err)
	is.Equal(calls, 1) // calls
}

func TestQuery(t *testing.T) {
	is := is.New(t)

	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		query := r.FormValue("query")
		is.Equal(query, "query {}")
		is.Equal(r.FormValue("variables"), `{"username":"matryer"}`+"\n")
		_, err := io.WriteString(w, `{"data":{"query":{"value":"some data"}}}`)
		is.NoErr(err)
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	client := newClient(srv.URL, UseMultipartForm())

	req := NewTestRequest("query {}")
	req.vars["username"] = "matryer"

	// check variables
	is.True(req != nil)
	is.Equal(req.vars["username"], "matryer")

	var resp struct {
		Value string
	}
	err := client.RunWithContext(ctx, req, &resp)
	is.NoErr(err)
	is.Equal(calls, 1)

	is.Equal(resp.Value, "some data")

}

func TestFile(t *testing.T) {
	is := is.New(t)

	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		file, header, err := r.FormFile("0")
		is.NoErr(err)
		defer file.Close()
		is.Equal(header.Filename, "filename.txt")

		b, err := ioutil.ReadAll(file)
		is.NoErr(err)
		is.Equal(string(b), `This is a file`)

		_, err = io.WriteString(w, `{"data":{"value":"some data"}}`)
		is.NoErr(err)
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	client := newClient(srv.URL, UseMultipartForm())
	f := strings.NewReader(`This is a file`)
	req := NewTestRequestWithFiles("query {}")
	req.files = append(req.files, File{Field: "file", Name: "filename.txt", R: f})
	var resp string
	err := client.RunWithContext(ctx, req, &resp)
	is.NoErr(err)
}

type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
