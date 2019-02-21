package model

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/inventory"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_client/inventory_operations"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/go-openapi/strfmt"
)

var (
	FailPlatforms = failures.Type("model.fail.platforms", api.FailUnknown)
)

var platformCache []*inventory_models.Platform

func FetchPlatforms() ([]*inventory_models.Platform, *failures.Failure) {
	if platformCache == nil {
		client := inventory.Get()

		response, err := client.Platforms(inventory_operations.NewPlatformsParams())
		if err != nil {
			return nil, FailPlatforms.Wrap(err)
		}

		platformCache = response.Payload
	}

	return platformCache, nil
}

func FetchPlatformByUID(uid strfmt.UUID) (*inventory_models.Platform, *failures.Failure) {
	platforms, fail := FetchPlatforms()
	if fail != nil {
		return nil, fail
	}

	for _, platform := range platforms {
		if platform.PlatformID != nil && *platform.PlatformID == uid {
			return platform, nil
		}
	}

	return nil, nil
}
