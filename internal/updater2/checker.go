package updater2

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/httpreq"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type Checker struct {
	apiURL         string
	currentChannel string
	currentVersion string
	httpreq        *httpreq.Client
}

var DefaultChecker = NewChecker(constants.APIUpdateURL, constants.BranchName, constants.Version)

func NewChecker(apiURL, currentChannel, currentVersion string) *Checker {
	return &Checker{
		apiURL,
		currentChannel,
		currentVersion,
		httpreq.New(),
	}
}

func (u *Checker) Check() (*AvailableUpdate, error) {
	return u.CheckFor("", "")
}

func (u *Checker) CheckFor(desiredChannel, desiredVersion string) (*AvailableUpdate, error) {
	platform := runtime.GOOS + "-" + runtime.GOARCH
	if desiredChannel == "" {
		desiredChannel = u.currentChannel
	}
	versionPath := ""
	if desiredVersion != "" {
		versionPath = "/" + desiredVersion
	}
	url := fmt.Sprintf("%s/%s/%s%s/info.json", u.apiURL, desiredChannel, versionPath, platform)
	res, err := u.httpreq.Get(url)
	if err != nil {
		return nil, errs.Wrap(err, "Could not fetch update info from %s", url)
	}

	info := &AvailableUpdate{}
	if err := json.Unmarshal(res, &info); err != nil {
		return nil, errs.Wrap(err, "Could not unmarshal update info: %s", res)
	}

	if info.channel == u.currentChannel && info.version == u.currentVersion {
		return nil, nil
	}

	info.url = u.apiURL + info.path

	return info, nil
}

// PrintUpdateMessage will print a message to stdout when an update is available.
// This will only print the message if the current project has a version lock AND if an update is available
func (u *Checker) PrintUpdateMessage(pjPath string, out output.Outputer) {
	if versionInfo, _ := projectfile.ParseVersionInfo(pjPath); versionInfo == nil {
		return
	}

	fmt.Println("checking v")
	info, err := u.Check()
	if err != nil {
		fmt.Printf("could not check for updates %v\n", err)
		logging.Error("Could not check for updates: %v", err)
		return
	}
	fmt.Printf("%v\n", info)

	if info != nil && info.Version() != constants.Version {
		out.Notice(output.Heading(locale.Tl("update_available_title", "Update Available")))
		out.Notice(locale.Tr("update_available", constants.Version, info.Version()))
	}
}
