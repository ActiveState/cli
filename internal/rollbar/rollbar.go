package rollbar

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/instanceid"
	"github.com/ActiveState/cli/internal/logging"
	configMediator "github.com/ActiveState/cli/internal/mediators/config"
	"github.com/ActiveState/cli/internal/singleton/uniqid"
	"github.com/rollbar/rollbar-go"
)

type config interface {
	GetBool(key string) bool
	Closed() bool
}

type doNotReport []string

func (d *doNotReport) Add(msg string) {
	if msg == "" || strings.TrimSpace(msg) == "" {
		return
	}

	for _, m := range *d {
		if m == msg {
			return
		}
	}

	*d = append(*d, msg)
}

func (d doNotReport) Contains(msg string) bool {
	for _, m := range d {
		if strings.EqualFold(m, msg) {
			return true
		}
	}

	return false
}

var (
	currentCfg          config
	reportingDisabled   bool
	DoNotReportMessages doNotReport
)

func readConfig() {
	reportingDisabled = currentCfg != nil && !currentCfg.Closed() && !currentCfg.GetBool(constants.ReportErrorsConfig)
	logging.Debug("Sending Rollbar reports? %v", reportingDisabled)
}

func init() {
	configMediator.RegisterOption(constants.ReportErrorsConfig, configMediator.Bool, true)
	configMediator.AddListener(constants.ReportErrorsConfig, readConfig)
}

// CurrentCmd holds the value of the current command being invoked
// it's a quick hack to allow us to log the command to rollbar without risking exposing sensitive info
var CurrentCmd string

func SetupRollbar(token string) {
	defer func() { handlePanics(recover()) }()
	// set user to unknown (if it has not been set yet)
	if _, ok := rollbar.Custom()["UserID"]; !ok {
		UpdateRollbarPerson("unknown", "unknown", "unknown")
	}
	rollbar.SetRetryAttempts(0)
	rollbar.SetToken(token)
	rollbar.SetEnvironment(constants.ChannelName)

	rollbar.SetCodeVersion(constants.Version)
	rollbar.SetServerRoot("github.com/ActiveState/cli")
	rollbar.SetLogger(&rollbar.SilentClientLogger{})

	// We can't use runtime.GOOS for the official platform field because rollbar sees that as a server-only platform
	// (which we don't have credentials for). So we're faking it with a custom field untill rollbar gets their act together.
	rollbar.SetPlatform("client")
	rollbar.SetTransform(func(data map[string]interface{}) {
		// We're not a server, so don't send server info (could contain sensitive info, like hostname)
		data["server"] = map[string]interface{}{}
		data["platform_os"] = runtime.GOOS
		if _, exists := data["request"]; !exists {
			data["request"] = map[string]string{}
		}
		if request, ok := data["request"].(map[string]string); ok {
			request["user_ip"] = "$remote_ip" // ask Rollbar to log the user's IP
		}
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

func SetConfig(cfg config) {
	currentCfg = cfg
	readConfig()
}

func UpdateRollbarPerson(userID, username, email string) {
	defer func() { handlePanics(recover()) }()
	rollbar.SetPerson(uniqid.Text(), username, email)

	custom := rollbar.Custom()
	if custom == nil { // could be nil if in tests that don't call SetupRollbar
		custom = map[string]interface{}{}
	}

	custom["UserID"] = userID
	custom["DeviceID"] = uniqid.Text()

	rollbar.SetCustom(custom)
}

// Wait is a wrapper around rollbar.Wait().
func Wait() { rollbar.Wait() }

var logDataAmenders []func(string) string

// AddLogDataAmender routes log data to be sent to Rollbar through the given function first.
// For example, that function might add more log data to be sent.
func AddLogDataAmender(f func(string) string) {
	logDataAmenders = append(logDataAmenders, f)
}

func logToRollbar(critical bool, message string, args ...interface{}) {
	// only log to rollbar when on release, beta or unstable channel and when built via CI (ie., non-local build)
	isPublicChannel := constants.ChannelName == constants.ReleaseChannel || constants.ChannelName == constants.BetaChannel || constants.ChannelName == constants.ExperimentalChannel
	if !isPublicChannel || !condition.BuiltViaCI() || reportingDisabled {
		return
	}

	data := map[string]interface{}{}
	logData := logging.ReadTail()
	if len(logData) == logging.TailSize {
		logData = "<truncated>\n" + logData
	}
	for _, f := range logDataAmenders {
		logData = f(logData)
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

	// Unpack error objects.
	for i, arg := range args {
		if err, ok := arg.(error); ok {
			args[i] = errs.JoinMessage(err)
		}
	}

	rollbarMsg := fmt.Sprintf("%s %s: %s", exec, flags, fmt.Sprintf(message, args...))
	if len(rollbarMsg) > 1000 {
		rollbarMsg = rollbarMsg[0:1000] + " <truncated>"
	}

	if DoNotReportMessages.Contains(rollbarMsg) {
		return
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
