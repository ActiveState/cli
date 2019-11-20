package logging

import (
	"log"
	"runtime"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/denisbrodbeck/machineid"
	"github.com/rollbar/rollbar-go"
)

func SetupRollbar() {
	UpdateRollbarPerson("unknown", "unknown", "unknown") // call again at authentication
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

func UpdateRollbarPerson(userID, username, email string) {
	machID, err := machineid.ID()
	if err != nil {
		Error("Cannot retrieve machine ID: %s", err.Error())
		machID = "unknown"
	}

	// MachineID is the only thing we have that is consistent between authed and unauthed users, so
	// we set that as the "person ID" in rollbar so the segmenting of data is consistent
	rollbar.SetPerson(machID, username, email)
	rollbar.SetCustom(map[string]interface{}{
		"UserID": userID,
	})
}
