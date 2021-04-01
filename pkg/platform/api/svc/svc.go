package svc

import (
	"fmt"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/gqlclient"
	"github.com/ActiveState/cli/internal/locale"
)

// New will create a new API client using default settings (for an authenticated version use the NewWithAuth version)
func New(cfg *config.Instance) (*gqlclient.Client, error) {
	port := cfg.GetInt(constants.SvcConfigPort)
	if port <= 0 {
		return nil, locale.NewError("err_svc_no_port", "The State Tool service does not appear to be running (no local port was configured).")
	}
	return gqlclient.New(fmt.Sprintf("http://127.0.0.1:%d/query", port), 0), nil
}
