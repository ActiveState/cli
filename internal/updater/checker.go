package updater

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"runtime"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/httpreq"
	"github.com/ActiveState/cli/internal/logging"
)

type httpGetter interface {
	Get(string) ([]byte, int, error)
}

type Configurable interface {
	GetString(string) string
}

type InvocationSource string

var InvocationSourceInstall InvocationSource = "install"
var InvocationSourceUpdate InvocationSource = "update"

type Checker struct {
	cfg            Configurable
	apiInfoURL     string
	fileURL        string
	currentChannel string
	currentVersion string
	httpreq        httpGetter

	InvocationSource InvocationSource
	VerifyVersion    bool
}

func NewDefaultChecker(cfg Configurable) *Checker {
	infoURL := constants.APIUpdateInfoURL
	if url, ok := os.LookupEnv("_TEST_UPDATE_INFO_URL"); ok {
		infoURL = url
	}
	updateURL := constants.APIUpdateURL
	if url, ok := os.LookupEnv("_TEST_UPDATE_URL"); ok {
		updateURL = url
	}
	return NewChecker(cfg, infoURL, updateURL, constants.BranchName, constants.Version, httpreq.New())
}

func NewChecker(cfg Configurable, infoURL, fileURL, currentChannel, currentVersion string, httpget httpGetter) *Checker {
	return &Checker{
		cfg,
		infoURL,
		fileURL,
		currentChannel,
		currentVersion,
		httpget,
		InvocationSourceUpdate,
		os.Getenv(constants.ForceUpdateEnvVarName) != "true",
	}
}

func (u *Checker) Check() (*AvailableUpdate, error) {
	return u.CheckFor(os.Getenv(constants.UpdateBranchEnvVarName), "")
}

func (u *Checker) CheckFor(desiredChannel, desiredVersion string) (*AvailableUpdate, error) {
	info, err := u.GetUpdateInfo(desiredChannel, desiredVersion)
	if err != nil {
		return nil, errs.Wrap(err, "Failed to get update info")
	}

	if info == nil || (u.VerifyVersion && info.Channel == u.currentChannel && info.Version == u.currentVersion) {
		return nil, nil
	}

	return info, nil
}

func (u *Checker) infoURL(tag, desiredVersion, branchName, platform string) string {
	v := make(url.Values)
	v.Set("channel", branchName)
	v.Set("platform", platform)
	v.Set("source", string(u.InvocationSource))

	if desiredVersion != "" {
		v.Set("target-version", desiredVersion)
	}

	if tag != "" {
		v.Set("tag", tag)
	}

	return u.apiInfoURL + "/info?" + v.Encode()
}

func (u *Checker) GetUpdateInfo(desiredChannel, desiredVersion string) (*AvailableUpdate, error) {
	if desiredChannel == "" {
		if overrideBranch := os.Getenv(constants.UpdateBranchEnvVarName); overrideBranch != "" {
			desiredChannel = overrideBranch
		} else {
			desiredChannel = u.currentChannel
		}
	}

	tag := u.cfg.GetString(CfgUpdateTag)
	infoURL := u.infoURL(tag, desiredVersion, desiredChannel, runtime.GOOS)
	logging.Debug("Getting update info: %s", infoURL)
	fmt.Printf("\n\nGetting update info: %s\n\n", infoURL)
	res, code, err := u.httpreq.Get(infoURL)
	if err != nil {
		if code == 404 || strings.Contains(string(res), "Could not retrieve update info") {
			// The above string match can be removed once https://www.pivotaltracker.com/story/show/179426519 is resolved
			logging.Debug("Update info 404s: %v", errs.JoinMessage(err))
			return nil, nil
		}
		return nil, errs.Wrap(err, "Could not fetch update info from %s", infoURL)
	}

	info := &AvailableUpdate{}
	if err := json.Unmarshal(res, &info); err != nil {
		return nil, errs.Wrap(err, "Could not unmarshal update info: %s", res)
	}

	info.url = u.fileURL + "/" + info.Path

	return info, nil
}
