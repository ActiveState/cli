package test

import (
	"fmt"
	"github.com/davidji99/rollrest-go/rollrest"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestListTeams(t *testing.T) {
	teardown := setup()
	defer teardown()

	mux.HandleFunc("/teams", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, getFixture("teams_list.json"))
	})

	team, r, err := client.Teams.List()

	assert.Nil(t, err)
	assert.Equal(t, "GET", r.RequestMethod)
	assert.Equal(t, 2, len(team.Result))
	assert.Equal(t, int64(123), team.Result[0].GetID())
	assert.Equal(t, int64(999), team.Result[0].GetAccountID())
	assert.Equal(t, "Everyone", team.Result[0].GetName())
	assert.Equal(t, "everyone", team.Result[0].GetAccessLevel())
}

func TestCreateTeams(t *testing.T) {
	teardown := setup()
	defer teardown()

	mux.HandleFunc("/teams", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, getFixture("teams_response.json"))
	})

	opts := &rollrest.TeamRequest{
		Name:        "new_team",
		AccessLevel: "owner",
	}

	team, r, err := client.Teams.Create(opts)

	assert.Nil(t, err)
	assert.Equal(t, "POST", r.RequestMethod)
	assert.Equal(t, "{\"name\":\"new_team\",\"access_level\":\"owner\"}", r.RequestBody)
	assert.Equal(t, "Owners", team.Result.GetName())
}

func TestGetTeams(t *testing.T) {
	teardown := setup()
	defer teardown()

	mux.HandleFunc("/team/123", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, getFixture("teams_response.json"))
	})

	team, r, err := client.Teams.Get(123)

	assert.Nil(t, err)
	assert.Equal(t, "GET", r.RequestMethod)
	assert.Equal(t, "Owners", team.Result.GetName())
}

func TestDeleteTeams(t *testing.T) {
	teardown := setup()
	defer teardown()

	mux.HandleFunc("/team/123", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, getFixture("teams_response.json"))
	})

	r, err := client.Teams.Delete(123)

	assert.Nil(t, err)
	assert.Equal(t, "DELETE", r.RequestMethod)
	assert.Equal(t, 200, r.StatusCode)
}

func TestListTeamUsers(t *testing.T) {
	teardown := setup()
	defer teardown()

	mux.HandleFunc("/team/123/users", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, getFixture("teams_users_list.json"))
	})

	u, r, err := client.Teams.ListUsers(123)

	assert.Nil(t, err)
	assert.Equal(t, "GET", r.RequestMethod)
	assert.Equal(t, 2, len(u.Result))
}

func TestIsUserAMember_Valid(t *testing.T) {
	teardown := setup()
	defer teardown()

	mux.HandleFunc(`/team/123/user/789`, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, getFixture("teams_users_get.json"))
	})

	isMember, r, err := client.Teams.IsUserMember(123, 789)

	assert.Nil(t, err)
	assert.Equal(t, "GET", r.RequestMethod)
	assert.Equal(t, true, isMember)
}

func TestIsUserAMember_Invalid(t *testing.T) {
	teardown := setup()
	defer teardown()

	mux.HandleFunc(`/team/123/user/790`, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, getFixture("teams_users_get.json"))
	})

	isMember, r, err := client.Teams.IsUserMember(123, 789)

	assert.NotNil(t, err)
	assert.Equal(t, "GET", r.RequestMethod)
	assert.Equal(t, false, isMember)
}

func TestAddUserToTeam(t *testing.T) {
	teardown := setup()
	defer teardown()

	mux.HandleFunc(`/team/123/user/789`, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, getFixture("teams_users_get.json"))
	})

	added, r, err := client.Teams.AddUser(123, 789)
	assert.Nil(t, err)
	assert.Equal(t, "PUT", r.RequestMethod)
	assert.Equal(t, true, added)
}

func TestRemoveUserToTeam(t *testing.T) {
	teardown := setup()
	defer teardown()

	mux.HandleFunc(`/team/123/user/789`, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, getFixture("teams_users_get.json"))
	})

	added, r, err := client.Teams.RemoveUser(123, 789)
	assert.Nil(t, err)
	assert.Equal(t, "DELETE", r.RequestMethod)
	assert.Equal(t, true, added)
}

func TestInviteUserToTeam(t *testing.T) {
	teardown := setup()
	defer teardown()

	mux.HandleFunc(`/team/123/invites`, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, getFixture("teams_invite_user.json"))
	})

	opts := &rollrest.TeamInviteRequest{Email: "user@example-company.com"}

	i, r, err := client.Teams.InviteUser(123, opts)

	assert.Nil(t, err)
	assert.Equal(t, "POST", r.RequestMethod)
	assert.Equal(t, int64(123), i.GetResult().GetID())
	assert.Equal(t, "user@example-company.com", i.GetResult().GetToEmail())
}

func TestListTeamInvites(t *testing.T) {
	teardown := setup()
	defer teardown()

	mux.HandleFunc(`/team/123/invites`, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, getFixture("teams_invites.json"))
	})

	i, r, err := client.Teams.ListInvites(123)

	assert.Nil(t, err)
	assert.Equal(t, "GET", r.RequestMethod)
	assert.Equal(t, 1, len(i.Result))
	assert.Equal(t, "user@example-company.com", i.Result[0].GetToEmail())
}

func TestListTeamProjects(t *testing.T) {
	teardown := setup()
	defer teardown()

	mux.HandleFunc(`/team/123/projects`, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, getFixture("teams_list_projects.json"))
	})

	p, r, err := client.Teams.ListProjects(123)

	assert.Nil(t, err)
	assert.Equal(t, "GET", r.RequestMethod)
	assert.Equal(t, 2, len(p.Result))
	assert.Equal(t, int64(123), p.Result[0].GetTeamID())
}

func TestTeamAssignProject(t *testing.T) {
	teardown := setup()
	defer teardown()

	mux.HandleFunc(`/team/123/project/456`, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, getFixture("team_project.json"))
	})

	p, r, err := client.Teams.AssignProject(123, 456)

	assert.Nil(t, err)
	assert.Equal(t, "PUT", r.RequestMethod)
	assert.Equal(t, int64(456), p.GetResult().GetProjectID())
}

func TestTeamRemoveProject(t *testing.T) {
	teardown := setup()
	defer teardown()

	mux.HandleFunc(`/team/123/project/456`, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, getFixture("team_project.json"))
	})

	r, err := client.Teams.RemoveProject(123, 456)

	assert.Nil(t, err)
	assert.Equal(t, "DELETE", r.RequestMethod)
}

func TestTeamHasProject(t *testing.T) {
	teardown := setup()
	defer teardown()

	mux.HandleFunc(`/team/123/project/456`, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, getFixture("team_project.json"))
	})

	valid, r, err := client.Teams.HasProject(123, 456)
	assert.Nil(t, err)
	assert.Equal(t, "GET", r.RequestMethod)
	assert.Equal(t, true, valid)
}
