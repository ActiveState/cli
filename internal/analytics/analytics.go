package analytics

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	ga "github.com/ActiveState/go-ogle-analytics"
)

var client *ga.Client

// CustomDimensions represents the custom dimensions sent with each event
var CustomDimensions *customDimensions

// CatRunCmd is the event category used for running commands
const CatRunCmd = "run-command"

// CatBuild is the event category used for headchef builds
const CatBuild = "build"

// ActBuildProject is the event action for requesting a build for a specific project
const ActBuildProject = "project"

type customDimensions struct {
	version    string
	branchName string
	userID     string
	output     string
}

func (d *customDimensions) SetOutput(output string) {
	d.output = output
}

func (d *customDimensions) toMap() map[string]string {
	return map[string]string{
		// Commented out idx 1 so it's clear why we start with 2. We used to log the hostname while dogfooding internally.
		// "1": "hostname (deprected)"
		"2": d.version,
		"3": d.branchName,
		"4": d.userID,
		"5": d.output,
	}
}

func init() {
	setup()
}

func setup() {
	id := logging.UniqID()
	var err error
	client, err = ga.NewClient(constants.AnalyticsTrackingID)
	if err != nil {
		logging.Error("Cannot initialize analytics: %s", err.Error())
		return
	}

	var userIDString string
	userID := authentication.Get().UserID()
	if userID != nil {
		userIDString = userID.String()
	}

	CustomDimensions = &customDimensions{
		version:    constants.Version,
		branchName: constants.BranchName,
		userID:     userIDString,
	}

	client.ClientID(id)

	if id == "unknown" {
		Event("error", "unknown machine id")
	}

	setUserAgentOverride(client)
}

// Event logs an event to google analytics
func Event(category string, action string) {
	go event(category, action)
}

func event(category string, action string) error {
	if client == nil || condition.InTest() {
		return nil
	}
	client.CustomDimensionMap(CustomDimensions.toMap())

	logging.Debug("Event: %s, %s", category, action)
	if category == CatRunCmd {
		client.Send(ga.NewPageview())
	}
	return client.Send(ga.NewEvent(category, action))
}

// EventWithLabel logs an event with a label to google analytics
func EventWithLabel(category string, action string, label string) {
	go eventWithLabel(category, action, label)
}

func eventWithLabel(category, action, label string) error {
	if client == nil || condition.InTest() {
		return nil
	}
	client.CustomDimensionMap(CustomDimensions.toMap())

	logging.Debug("Event+label: %s, %s, %s", category, action, label)
	return client.Send(ga.NewEvent(category, action).Label(label))
}

// EventWithValue logs an event with an integer value to google analytics
func EventWithValue(category string, action string, value int64) {
	go eventWithValue(category, action, value)
}

func eventWithValue(category string, action string, value int64) error {
	if client == nil || condition.InTest() {
		return nil
	}
	client.CustomDimensionMap(CustomDimensions.toMap())

	logging.Debug("Event+value: %s, %s, %s", category, action, value)
	return client.Send(ga.NewEvent(category, action).Value(value))
}

func setUserAgentOverride(client *ga.Client) {
	productName := "state"
	productVersion := constants.VersionNumber
	viewer := "compatible"
	opsysName := "Unknown"
	opsysVersion := "0.0"

	switch runtime.GOOS {
	case "linux":
		if _, ok := os.LookupEnv("DISPLAY"); ok {
			viewer = "X11"
		}

		opsysName = "Linux"

		if version, err := linuxVersion(); err == nil {
			opsysVersion = version
		}

	case "darwin":
		viewer = "Macintosh"

		opsysArch := "Intel"
		if strings.Contains(runtime.GOARCH, "ppc") {
			opsysArch = "PPC"
		}
		opsysName = fmt.Sprintf("%s %s", opsysArch, "Mac OS X")

		if version, err := macVersion(); err == nil {
			opsysVersion = version
		}

	case "windows":
		opsysName = "Windows NT"

		if version, err := windowsVersion(); err == nil {
			opsysVersion = version
		}
	}

	uaText := fmt.Sprintf(
		"%s/%s (%s; %s %s)",
		productName, productVersion,
		viewer, opsysName, opsysVersion,
	)

	client.UserAgentOverride(uaText)
}

// linuxVersion returns architecture (this is the data associated with the OS
// in Linux user-agent data)
func linuxVersion() (string, error) {
	archData, err := exec.Command("uname", "-i").Output()
	if err != nil {
		return "", nil
	}

	archData = bytes.TrimSpace(archData)
	if len(archData) == 0 {
		return "", errors.New("cannot parse linux version")
	}

	return string(archData), nil
}

// macVersion returns version as "10.15.1"
func macVersion() (string, error) {
	cmd := exec.Command("defaults", "read", "loginwindow", "SystemVersionStampAsString")
	versionData, err := cmd.Output()
	if err != nil {
		return "", err
	}

	versionData = bytes.TrimSpace(versionData)
	if len(versionData) == 0 {
		return "", errors.New("cannot parse mac version")
	}

	return string(versionData), nil
}

// windowsVersion returns version as "10.0"
func windowsVersion() (string, error) {
	data, err := exec.Command("wmic", "os", "get", "version").Output()
	if err != nil {
		return "", err
	}

	// command outputs multiple lines; version expected on second line
	data = bytes.Replace(data, []byte("\r\n"), []byte("\n"), -1)
	lines := bytes.Split(data, []byte("\n"))
	index := 1
	if len(lines) < 2 {
		index = 0
	}
	version := lines[index]

	// version format example: 10.0.2345
	vsplit := bytes.Split(version, []byte("."))
	ct := 2
	if len(vsplit) < 2 {
		ct = 1
	}

	out := bytes.TrimSpace(bytes.Join(vsplit[:ct], []byte(".")))
	if len(out) == 0 {
		return "", errors.New("cannot parse windows version")
	}

	return string(out), nil
}
