package updater2

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/httpreq"
)

type Checker struct {
	apiURL          string
	currentVersion  string
	desiredVersion  string
	channel         string
	httpreq         *httpreq.Client
	availableUpdate *Update
}

func NewChecker(apiURL string, currentVersion, desiredVersion, channel string) *Checker {
	return &Checker{
		apiURL,
		currentVersion,
		desiredVersion,
		channel,
		httpreq.New(),
		nil,
	}
}

func (u *Checker) Check(force bool) (*Update, error) {
	if !force && u.availableUpdate != nil {
		return u.availableUpdate, nil
	}

	platform := runtime.GOOS + "-" + runtime.GOARCH
	url := fmt.Sprintf("%s/%s/%s/info.json", u.apiURL, u.channel, platform)
	res, err := u.httpreq.Get(url)
	if err != nil {
		return nil, errs.Wrap(err, "Could not fetch update info from %s", url)
	}

	info := &Update{}
	if err := json.Unmarshal(res, &info); err != nil {
		return nil, errs.Wrap(err, "Could not unmarshal update info: %s", res)
	}

	info.url = u.apiURL + info.path

	return info, nil
}
