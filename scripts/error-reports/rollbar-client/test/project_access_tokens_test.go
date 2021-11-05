package test

import (
	"fmt"
	"github.com/davidji99/rollrest-go/rollrest"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestListProjectAccessTokens(t *testing.T) {
	teardown := setup()
	defer teardown()

	mux.HandleFunc("/project/123/access_tokens", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, getFixture("pat_list.json"))
	})

	p, r, err := client.ProjectAccessTokens.List(123)

	assert.Nil(t, err)
	assert.Equal(t, "GET", r.RequestMethod)
	assert.Equal(t, 2, len(p.Result))
	assert.Equal(t, "automation", p.Result[0].GetName())
	assert.Equal(t, "write", p.Result[0].Scopes[0])
}

func TestGetProjectAccessTokens_Valid(t *testing.T) {
	teardown := setup()
	defer teardown()

	mux.HandleFunc("/project/123/access_tokens", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, getFixture("pat_list.json"))
	})

	p, r, err := client.ProjectAccessTokens.Get(123, "abcdefg")

	assert.Nil(t, err)
	assert.Equal(t, "GET", r.RequestMethod)
	assert.Equal(t, "automation", p.GetName())
	assert.Equal(t, "write", p.Scopes[0])
}

func TestGetProjectAccessTokens_Invalid(t *testing.T) {
	teardown := setup()
	defer teardown()

	mux.HandleFunc("/project/123/access_tokens", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, getFixture("pat_list.json"))
	})

	_, r, err := client.ProjectAccessTokens.Get(123, "zzzzzz")

	assert.Equal(t, "specified project access token not found", err.Error())
	assert.Equal(t, "GET", r.RequestMethod)
}

func TestCreateProjectAccessTokens(t *testing.T) {
	teardown := setup()
	defer teardown()

	mux.HandleFunc("/project/123/access_tokens", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, getFixture("pat_modify.json"))
	})

	opts := &rollrest.PATCreateRequest{
		Name:                 "new automation token",
		Scopes:               []string{"read", "write"},
		Status:               "enabled",
		RateLimitWindowSize:  60,
		RateLimitWindowCount: 1500,
	}

	p, r, err := client.ProjectAccessTokens.Create(123, opts)

	assert.Nil(t, err)
	assert.Equal(t, "POST", r.RequestMethod)
	assert.Equal(t, "{\"name\":\"new automation token\",\"scopes\":[\"read\",\"write\"],\"status\":\"enabled\",\"rate_limit_window_size\":60,\"rate_limit_window_count\":1500}", r.RequestBody)
	assert.Equal(t, "new automation token", p.GetResult().GetName())
}

func TestUpdateProjectAccessTokens(t *testing.T) {
	teardown := setup()
	defer teardown()

	mux.HandleFunc("/project/123/access_token/abcdefg", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, getFixture("pat_modify.json"))
	})

	opts := &rollrest.PATUpdateRequest{
		RateLimitWindowSize:  60,
		RateLimitWindowCount: 1500,
	}

	p, r, err := client.ProjectAccessTokens.Update(123, "abcdefg", opts)

	assert.Nil(t, err)
	assert.Equal(t, "PATCH", r.RequestMethod)
	assert.Equal(t, "{\"rate_limit_window_size\":60,\"rate_limit_window_count\":1500}", r.RequestBody)
	assert.Equal(t, 1500, p.GetResult().GetRateLimitWindowCount())
}
