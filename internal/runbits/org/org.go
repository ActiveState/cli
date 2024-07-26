package org

import (
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

var ErrNoOwner = errs.New("Could not find organization")

type ErrOwnerNotFound struct {
	DesiredOwner string
}

func (e ErrOwnerNotFound) Error() string {
	return "could not find this organization"
}

type configurer interface {
	GetString(string) string
}

// Get returns the name of an organization/owner in order of precedence:
// - Returns the normalized form of the given org name.
// - Returns the name of the most recently used org.
// - Returns the name of the first org you are a member of.
func Get(desiredOrg string, auth *authentication.Auth, cfg configurer) (string, error) {
	orgs, err := model.FetchOrganizations(auth)
	if err != nil {
		return "", errs.Wrap(err, "Unable to get the user's writable orgs")
	}

	// Prefer the desired org if it's valid
	if desiredOrg != "" {
		// Match the case of the organization.
		// Otherwise the incorrect case will be written to the project file.
		for _, org := range orgs {
			if strings.EqualFold(org.URLname, desiredOrg) {
				return org.URLname, nil
			}
		}
		return "", &ErrOwnerNotFound{desiredOrg}
	}

	// Use the last used namespace if it's valid
	lastUsed := cfg.GetString(constants.LastUsedNamespacePrefname)
	if lastUsed != "" {
		ns, err := project.ParseNamespace(lastUsed)
		if err != nil {
			return "", errs.Wrap(err, "Unable to parse last used namespace")
		}

		for _, org := range orgs {
			if strings.EqualFold(org.URLname, ns.Owner) {
				return org.URLname, nil
			}
		}
	}

	// Use the first org if there is one
	if len(orgs) > 0 {
		return orgs[0].URLname, nil
	}

	return "", ErrNoOwner
}
