package test

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestListUsers(t *testing.T) {
	teardown := setup()
	defer teardown()

	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, getFixture("users_list.json"))
	})

	u, r, err := client.Users.List()

	assert.Nil(t, err)
	assert.Equal(t, "GET", r.RequestMethod)
	assert.Equal(t, 3, len(u.Result.Users))
	assert.Equal(t, "username1@email.com", u.Result.Users[0].GetEmail())
	assert.Equal(t, int64(123), u.Result.Users[0].GetID())
	assert.Equal(t, "username1", u.Result.Users[0].GetUsername())
}

func TestGetUsers(t *testing.T) {
	teardown := setup()
	defer teardown()

	mux.HandleFunc("/user/123", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, getFixture("users_get.json"))
	})

	u, r, err := client.Users.Get(123)

	assert.Nil(t, err)
	assert.Equal(t, "GET", r.RequestMethod)
	assert.Equal(t, "username1@email.com", u.Result.GetEmail())
	assert.Equal(t, int64(123), u.Result.GetID())
	assert.Equal(t, "username1", u.Result.GetUsername())
	assert.Equal(t, 1, u.Result.GetEmailEnabled())
}

func TestListUserTeams(t *testing.T) {
	teardown := setup()
	defer teardown()

	mux.HandleFunc("/user/123/teams", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, getFixture("users_list_teams.json"))
	})

	teams, r, err := client.Users.ListTeams(123)

	assert.Nil(t, err)
	assert.Equal(t, "GET", r.RequestMethod)
	assert.Equal(t, "Owners", teams.GetResult().Teams[0].GetName())
	assert.Equal(t, int64(123), teams.GetResult().Teams[0].GetID())
	assert.Equal(t, "owner", teams.GetResult().Teams[0].GetAccessLevel())
	assert.Equal(t, int64(456), teams.GetResult().Teams[0].GetAccountID())
}

func TestListUserProjects(t *testing.T) {
	teardown := setup()
	defer teardown()

	mux.HandleFunc("/user/123/projects", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, getFixture("users_list_projects.json"))
	})

	p, r, err := client.Users.ListProjects(123)

	assert.Nil(t, err)
	assert.Equal(t, "GET", r.RequestMethod)
	assert.Equal(t, "project_one", p.GetResult().Projects[0].GetSlug())
	assert.Equal(t, 1, p.GetResult().Projects[0].GetStatus())
	assert.Equal(t, int64(991), p.GetResult().Projects[0].GetID())
	assert.Equal(t, int64(123), p.GetResult().Projects[0].GetAccountID())
}
