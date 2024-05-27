package model

import (
	"github.com/ActiveState/cli/internal/errs"
	mono_models "github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

// FetchTiers fetachs the tier for the given organization
func FetchTiers(auth *authentication.Auth) ([]*mono_models.Tier, error) {
	authClient, err := auth.Client()
	if err != nil {
		return nil, errs.Wrap(err, "Could not get auth client")
	}
	res, err := authClient.Tiers.GetTiers(nil, auth.ClientAuth())

	if err != nil {
		return nil, processOrgErrorResponse(err)
	}

	return res.Payload, nil
}
