package logging

import (
	"fmt"
	"runtime"
	"time"

	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/rollbar/rollbar-go"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/machineid"
)

func SetupRollbar(token string) {
	// set user to unknown (if it has not been set yet)
	if _, ok := rollbar.Custom()["UserID"]; !ok {
		UpdateRollbarPerson("unknown", "unknown", "unknown")
	}
	rollbar.SetToken(token)
	rollbar.SetEnvironment(constants.BranchName)

	dateTime := constants.Date
	t, err := time.Parse(constants.DateTimeFormatRecord, constants.Date)
	if err == nil {
		dateTime = t.Format("2006-01-02T15:04:05-0700") // ISO 8601
	}

	rollbar.SetCodeVersion(fmt.Sprintf("%s-%s", dateTime, constants.RevisionHashShort))
	rollbar.SetServerRoot("github.com/ActiveState/cli")
	rollbar.SetLogger(&rollbar.SilentClientLogger{})

	// We can't use runtime.GOOS for the official platform field because rollbar sees that as a server-only platform
	// (which we don't have credentials for). So we're faking it with a custom field untill rollbar gets their act together.
	rollbar.SetPlatform("client")
	rollbar.SetTransform(func(data map[string]interface{}) {
		// We're not a server, so don't send server info (could contain sensitive info, like hostname)
		data["server"] = map[string]interface{}{}
		data["platform_os"] = runtime.GOOS
	})

	source, err := storage.InstallSource()
	if err != nil {
		rollbar.Log("error", err.Error())
	}

	rollbar.SetCustom(map[string]interface{}{
		"install_source": source,
	})
}

func UpdateRollbarPerson(userID, username, email string) {
	machID := machineid.UniqID()

	// MachineID is the only thing we have that is consistent between authed and unauthed users, so
	// we set that as the "person ID" in rollbar so the segmenting of data is consistent
	rollbar.SetPerson(machID, username, email)

	custom := rollbar.Custom()
	if custom == nil { // could be nil if in tests that don't call SetupRollbar
		custom = map[string]interface{}{}
	}

	custom["UserID"] = userID
	rollbar.SetCustom(custom)
}
