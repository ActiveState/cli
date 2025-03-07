package gqlclient

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/matryer/is"
)

func TestDoJSON(t *testing.T) {
	is := is.New(t)
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		is.Equal(r.Method, http.MethodPost)
		b, err := ioutil.ReadAll(r.Body)
		is.NoErr(err)
		is.Equal(string(b), `{"query":"query {}","variables":null}`+"\n")
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
	client := newClient(srv.URL)

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	var responseData map[string]interface{}
	err := client.RunWithContext(ctx, &TestRequest{"query {}", nil, nil}, &responseData)
	is.NoErr(err)
	is.Equal(calls, 1) // calls
	is.Equal(responseData["something"], "yes")
}

func TestQueryJSON(t *testing.T) {
	is := is.New(t)

	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		b, err := ioutil.ReadAll(r.Body)
		is.NoErr(err)
		is.Equal(string(b), `{"query":"query {}","variables":{"username":"matryer"}}`+"\n")
		_, err = io.WriteString(w, `{"data":{"query":{"value":"some data"}}}`)
		is.NoErr(err)
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	client := newClient(srv.URL)

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

func TestHeader(t *testing.T) {
	is := is.New(t)

	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		is.Equal(r.Header.Get("X-Custom-Header"), "123")

		_, err := io.WriteString(w, `{
			"data": {
				"query": {
					"value": "some data"
				}
			}
		}`)
		is.NoErr(err)
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	client := newClient(srv.URL)

	req := NewTestRequest("query {}")
	req.headers["X-Custom-Header"] = []string{"123"}
	var resp struct {
		Value string
	}
	err := client.RunWithContext(ctx, req, &resp)
	is.NoErr(err)
	is.Equal(calls, 1)

	is.Equal(resp.Value, "some data")
}
