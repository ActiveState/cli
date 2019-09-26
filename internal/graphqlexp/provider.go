package main

import (
	"strings"

	"github.com/ActiveState/cli/internal/gqlclient"
)

// data structures

type user struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type allUsersResp struct {
	AllUsers []*user `json:"allUsers"`
}

// interface

type provider interface {
	allUsers() (*allUsersResp, error)
	allUsersByNameContains(string) (*allUsersResp, error)
}

func newProvider(isTest bool, endpoint string, hdr gqlclient.Header) provider {
	switch isTest {
	case true:
		return newGQLMock()
	default:
		return &gqlClient{gqlclient.New(endpoint, hdr)}
	}
}

// implementations (one real, one mock) - common methods near each other

type gqlClient struct {
	*gqlclient.GQLClient
}

var _ provider = (*gqlClient)(nil) // ensure *gqlClient implements provider

type gqlMock struct {
	aus *allUsersResp
}

var _ provider = (*gqlMock)(nil) // ensure *gqlMock implements provider

func newGQLMock() *gqlMock {
	us := []*user{
		{ID: "testerid", Name: "testername"},
		{ID: "xxxxid", Name: "xxxxname"},
	}

	return &gqlMock{aus: &allUsersResp{us}}
}

func (gqlc *gqlClient) allUsers() (*allUsersResp, error) {
	req := gqlc.NewRequest(`
query {
  allUsers {
    id
    name
  }
}
`)

	var resp allUsersResp
	err := gqlc.Run(req, &resp)
	return &resp, err
}

func (gm *gqlMock) allUsers() (*allUsersResp, error) {
	return gm.aus, nil
}

func (gqlc *gqlClient) allUsersByNameContains(val string) (*allUsersResp, error) {
	req := gqlc.NewRequest(`
query ($nameSubStr: String) {
  allUsers (filter: {name_contains: $nameSubStr}) {
    id
    name
  }
}
`)
	req.Var("nameSubStr", val)

	var resp allUsersResp
	err := gqlc.Run(req, &resp)
	return &resp, err
}

func (gm *gqlMock) allUsersByNameContains(val string) (*allUsersResp, error) {
	var us []*user
	for _, u := range gm.aus.AllUsers {
		if strings.Contains(u.Name, val) {
			us = append(us, u)
		}
	}

	resp := allUsersResp{
		AllUsers: us,
	}

	return &resp, nil
}
