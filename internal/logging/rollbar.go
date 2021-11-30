package logging

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/machineid"
	"github.com/ActiveState/cli/internal/singleton/uniqid"

	"github.com/rollbar/rollbar-go"
)

type RollbarLogger struct{}

func (s *RollbarLogger) Printf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

type RollbarErrorLogger struct{
	reporter func(string)
}

func (s *RollbarErrorLogger) Printf(format string, args ...interface{}) {
	if ! strings.HasPrefix(format, "Rollbar") { // All rollbar errors I observed are prefixed with "Rollbar"
		return
	}

	s.reporter(fmt.Sprintf(format, args...))
}

func SetupRollbarReporter(reporter func(string)) {
	rollbar.SetLogger(&RollbarErrorLogger{reporter})
}

func SetupRollbar(token string) {
	defer handlePanics(recover())
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
