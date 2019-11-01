package logging

import (
	"log"
	"runtime"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/denisbrodbeck/machineid"
	"github.com/rollbar/rollbar-go"
)

func SetupRollbar() {
	UpdateRollbarPerson("N/A") // call again at authentication
	rollbar.SetToken(constants.RollbarToken)
	rollbar.SetEnvironment(constants.BranchName)
	rollbar.SetCodeVersion(constants.RevisionHash)
	rollbar.SetServerRoot("github.com/ActiveState/cli")

	// We can't use runtime.GOOS for the official platform field because rollbar sees that as a server-only platform
	// (which we don't have credentials for). So we're faking it with a custom field untill rollbar gets their act together.
	rollbar.SetPlatform("client")
	rollbar.SetTransform(func(data map[string]interface{}) {
		// We're not a server, so don't send server info (could contain sensitive info, like hostname)
		data["server"] = map[string]interface{}{}
		data["platform_os"] = runtime.GOOS
	})

	log.SetOutput(CurrentHandler().Output())
}

func UpdateRollbarPerson(userID string) {
	machID, err := machineid.ID()
	if err != nil {
		Error("Cannot retrieve machine ID: %s", err.Error())
		machID = "unknown"
	}
	rollbar.SetPerson(machID, userID, machID)
}
