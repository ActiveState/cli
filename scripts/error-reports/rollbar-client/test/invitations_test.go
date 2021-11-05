package test

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestGetInvitation(t *testing.T) {
	teardown := setup()
	defer teardown()

	mux.HandleFunc("/invite/123", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, getFixture("invitations_get.json"))
	})

	p, r, err := client.Invitations.Get(123)

	assert.Nil(t, err)
	assert.Equal(t, "GET", r.RequestMethod)
	assert.Equal(t, "invited_user@email.com", p.GetResult().GetToEmail())
	assert.Equal(t, int64(999), p.GetResult().GetFromUserID())
}

func TestCancelInvitation(t *testing.T) {
	teardown := setup()
	defer teardown()

	mux.HandleFunc("/invite/123", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, getFixture("invitations_cancel.json"))
	})

	response, r, err := client.Invitations.Cancel(123)

	assert.Nil(t, err)
	assert.Equal(t, "DELETE", r.RequestMethod)
	assert.Equal(t, 0, response.GetErr())
}
