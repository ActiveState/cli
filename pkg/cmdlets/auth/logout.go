package auth

import (
	"context"

	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/svcmanager"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
)

func Logout(cfg configurable, mgr *svcmanager.Manager) error {
	auth := authentication.LegacyGet()
	auth.Logout()

	if err := keypairs.DeleteWithDefaults(cfg); err != nil {
		return err
	}

	svcmdl, err := model.NewSvcModel(context.Background(), cfg, mgr)
	if err != nil {
		logging.Error("Error notifying service of updated authentication (logout)")
	}

	logging.Debug("Sending Authentication Event (logout)")
	svcmdl.AuthenticationEvent(context.Background(), "")

	return nil
}
