package rollbar

import (
	"fmt"
	"os"
	"runtime"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/instanceid"
	"github.com/ActiveState/cli/internal/machineid"
	"github.com/ActiveState/cli/internal/singleton/uniqid"

	"github.com/rollbar/rollbar-go"
)

func SetupRollbar(token string) {
	defer handlePanics(recover())
	// set user to unknown (if it has not been set yet)
	if _, ok := rollbar.Custom()["UserID"]; !ok {
		UpdateRollbarPerson("unknown", "unknown", "unknown")
	}
	rollbar.SetRetryAttempts(0)
	rollbar.SetToken(token)
	rollbar.SetEnvironment(constants.BranchName)

	rollbar.SetCodeVersion(constants.Version)
	rollbar.SetServerRoot("github.com/ActiveState/cli")
	rollbar.SetLogger(&rollbar.SilentClientLogger{})
	rollbar.SetCaptureIp(rollbar.CaptureIpFull)

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
		"instance_id":    instanceid.ID(),
	})
}

func UpdateRollbarPerson(userID, username, email string) {
	defer handlePanics(recover())
	rollbar.SetPerson(uniqid.Text(), username, email)

	custom := rollbar.Custom()
	if custom == nil { // could be nil if in tests that don't call SetupRollbar
		custom = map[string]interface{}{}
	}

	custom["UserID"] = userID
	custom["MachineID"] = machineid.UniqID()

	rollbar.SetCustom(custom)
}

// Wait is a wrapper around rollbar.Wait().
func Wait() { rollbar.Wait() }

// Critical is a wrapper around rollbar.Critical().
func Critical(interfaces ...interface{}) { rollbar.Critical(interfaces...) }

// Error is a wrapper around rollbar.Error().
func Error(interfaces ...interface{}) { rollbar.Error(interfaces...) }

func handlePanics(err interface{}) {
	if err == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "Failed to log error. Please report this on the forums if it keeps happening. Error: %v\n", err)
}
