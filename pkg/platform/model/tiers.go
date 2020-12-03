package model

import (
	"github.com/ActiveState/cli/internal/failures"
	mono_models "github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

// FetchTiers fetachs the tier for the given organization
func FetchTiers() ([]*mono_models.Tier, error) {
	res, err := authentication.Client().Tiers.GetTiers(nil, authentication.ClientAuth())

	if err != nil {
		return nil, processOrgErrorResponse(err)
	}

	return res.Payload, nil
}
