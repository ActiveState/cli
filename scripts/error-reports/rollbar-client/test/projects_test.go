package test

import (
	"fmt"
	"github.com/davidji99/rollrest-go/rollrest"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestListProjects(t *testing.T) {
	teardown := setup()
	defer teardown()

	mux.HandleFunc("/projects", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, getFixture("projects_list.json"))
	})

	p, r, err := client.Projects.List()

	assert.Nil(t, err)
	assert.Equal(t, "GET", r.RequestMethod)
	assert.Equal(t, 1, len(p.Result))
	assert.Equal(t, int64(123), p.Result[0].GetID())
	assert.Equal(t, int64(999), p.Result[0].GetAccountID())
	assert.Equal(t, "my project", p.Result[0].GetName())
	assert.Equal(t, 1, p.Result[0].GetSettingsData().GetFingerprintVersions().GetAndroidAndroid())
}

func TestListAllProjects(t *testing.T) {
	teardown := setup()
	defer teardown()

	mux.HandleFunc("/projects", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, getFixture("projects_list.json"))
	})

	p, r, err := client.Projects.ListAll()

	assert.Nil(t, err)
	assert.Equal(t, "GET", r.RequestMethod)
	assert.Equal(t, 2, len(p.Result))
	assert.Equal(t, int64(123), p.Result[0].GetID())
	assert.Equal(t, int64(999), p.Result[0].GetAccountID())
	assert.Equal(t, "my project", p.Result[0].GetName())
	assert.Equal(t, 1, p.Result[0].GetSettingsData().GetFingerprintVersions().GetAndroidAndroid())
	assert.Equal(t, "", p.Result[1].GetName())
}

func TestGetProject(t *testing.T) {
	teardown := setup()
	defer teardown()

	mux.HandleFunc("/project/123", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, getFixture("projects_get.json"))
	})

	p, r, err := client.Projects.Get(123)

	assert.Nil(t, err)
	assert.Equal(t, "GET", r.RequestMethod)
	assert.Equal(t, int64(123), p.GetResult().GetID())
	assert.Equal(t, int64(999), p.GetResult().GetAccountID())
}

func TestCreateProject(t *testing.T) {
	teardown := setup()
	defer teardown()

	mux.HandleFunc("/projects", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, getFixture("projects_get.json"))
	})

	opts := &rollrest.ProjectRequest{Name: "my project"}

	p, r, err := client.Projects.Create(opts)

	assert.Nil(t, err)
	assert.Equal(t, "POST", r.RequestMethod)
	assert.Equal(t, "{\"name\":\"my project\"}", r.RequestBody)
	assert.Equal(t, "my project", p.GetResult().GetName())
}

func TestDeleteProject(t *testing.T) {
	teardown := setup()
	defer teardown()

	mux.HandleFunc("/project/123", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, getFixture("projects_get.json"))
	})

	r, err := client.Projects.Delete(123)

	assert.Nil(t, err)
	assert.Equal(t, "DELETE", r.RequestMethod)
}
