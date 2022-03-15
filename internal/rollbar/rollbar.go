package rollbar

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/instanceid"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/machineid"
	"github.com/ActiveState/cli/internal/singleton/uniqid"

	"github.com/rollbar/rollbar-go"
)

// CurrentCmd holds the value of the current command being invoked
// it's a quick hack to allow us to log the command to rollbar without risking exposing sensitive info
var CurrentCmd string

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

func logToRollbar(critical bool, message string, args ...interface{}) {
	// only log to rollbar when on release, beta or unstable branch and when built via CI (ie., non-local build)
	isPublicChannel := (constants.BranchName == constants.ReleaseBranch || constants.BranchName == constants.BetaBranch || constants.BranchName == constants.ExperimentalBranch)
	if !isPublicChannel || !condition.BuiltViaCI() {
		return
	}

	data := map[string]interface{}{}
	logData := logging.ReadTail()
	if len(logData) == logging.TailSize {
		logData = "<truncated>\n" + logData
	}
	data["log_file_data"] = logData

	exec := CurrentCmd
	if exec == "" {
		exec = strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe")
	}
	flags := []string{}
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "-") {
			idx := strings.Index(arg, "=")
			if idx != -1 {
				arg = arg[0:idx]
			}
			flags = append(flags, arg)
		}
	}

	rollbarMsg := fmt.Sprintf("%s %s: %s", exec, flags, fmt.Sprintf(message, args...))
	if len(rollbarMsg) > 1000 {
		rollbarMsg = rollbarMsg[0:1000] + " <truncated>"
	}

	if critical {
		rollbar.Critical(fmt.Errorf(rollbarMsg), data)
	} else {
		rollbar.Error(fmt.Errorf(rollbarMsg), data)
	}
}

// Critical logs a critical error to rollbar.
func Critical(message string, args ...interface{}) {
	logToRollbar(true, message, args...)
}

// Error logs an error to rollbar.
func Error(message string, args ...interface{}) {
	logToRollbar(false, message, args...)
}

func handlePanics(err interface{}) {
	if err == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "Failed to log error. Please report this on the forums if it keeps happening. Error: %v\n", err)
}
