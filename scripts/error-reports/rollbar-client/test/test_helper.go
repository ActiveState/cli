package test

import (
	"fmt"
	"github.com/davidji99/rollrest-go/rollrest"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
)

var (
	mux       *http.ServeMux
	server    *httptest.Server
	client    *rollrest.Client
	clientErr error
)

func setup() func() {
	mux = http.NewServeMux()
	server = httptest.NewServer(mux)

	fmt.Println(server.URL)
	client, clientErr = rollrest.New(rollrest.BaseURL(server.URL), rollrest.AuthAAT("account_access_token"),
		rollrest.AuthPAT("project_access_token"))
	if clientErr != nil {
		panic(clientErr)
	}

	return func() {
		server.Close()
	}
}

func getFixture(path string) string {
	b, err := ioutil.ReadFile("../testdata/fixtures/" + path)
	if err != nil {
		panic(err)
	}
	return string(b)
}
