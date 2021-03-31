package updater2

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/httpreq"
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
