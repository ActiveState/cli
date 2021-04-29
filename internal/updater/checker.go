package updater

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/httpreq"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type httpGetter interface {
	Get(string) ([]byte, error)
}

type Checker struct {
	apiURL         string
	currentChannel string
	currentVersion string
	httpreq        httpGetter
}

var DefaultChecker = newDefaultChecker()

func newDefaultChecker() *Checker {
	updateURL := constants.APIUpdateURL
	if url, ok := os.LookupEnv("_TEST_UPDATE_URL"); ok {
		updateURL = url
	}
	return NewChecker(updateURL, constants.BranchName, constants.Version, httpreq.New())
}

func NewChecker(apiURL, currentChannel, currentVersion string, httpget httpGetter) *Checker {
	return &Checker{
		apiURL,
		currentChannel,
		currentVersion,
		httpget,
	}
}

func (u *Checker) Check() (*AvailableUpdate, error) {
	return u.CheckFor("", "")
}

func (u *Checker) CheckFor(desiredChannel, desiredVersion string) (*AvailableUpdate, error) {
	info, err := u.GetUpdateInfo(desiredChannel, desiredVersion)
	if err != nil {
		return nil, errs.Wrap(err, "Failed to get update info")
	}

	if info.Channel == u.currentChannel && info.Version == u.currentVersion {
		return nil, nil
	}

	return info, nil
}

func (u *Checker) GetUpdateInfo(desiredChannel, desiredVersion string) (*AvailableUpdate, error) {
	platform := runtime.GOOS + "-" + runtime.GOARCH
	if desiredChannel == "" {
		if overrideBranch := os.Getenv(constants.UpdateBranchEnvVarName); overrideBranch != "" {
			desiredChannel = overrideBranch
		} else {
			desiredChannel = u.currentChannel
		}
	}
	versionPath := ""
	if desiredVersion != "" {
		versionPath = "/" + desiredVersion
	}
	url := fmt.Sprintf("%s%s%s/%s/info.json", u.apiURL, desiredChannel, versionPath, platform)
	res, err := u.httpreq.Get(url)
	if err != nil {
		return nil, errs.Wrap(err, "Could not fetch update info from %s", url)
	}

	info := &AvailableUpdate{}
	if err := json.Unmarshal(res, &info); err != nil {
		return nil, errs.Wrap(err, "Could not unmarshal update info: %s", res)
	}

	info.url = u.apiURL + info.Path

	return info, nil
}

// PrintUpdateMessage will print a message to stdout when an update is available.
// This will only print the message if the current project has a version lock AND if an update is available
func (u *Checker) PrintUpdateMessage(pjPath string, out output.Outputer) {
	if versionInfo, _ := projectfile.ParseVersionInfo(pjPath); versionInfo == nil {
		return
	}

	info, err := u.Check()
	if err != nil {
		logging.Error("Could not check for updates: %v", err)
		return
	}

	if info != nil && info.Version != constants.Version {
		out.Notice(output.Heading(locale.Tl("update_available_title", "Update Available")))
		out.Notice(locale.Tr("update_available", constants.Version, info.Version))
	}
}
