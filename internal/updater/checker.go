package updater

import (
	"encoding/json"
	"net/url"
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

type Configurable interface {
	GetString(string) string
}

type Checker struct {
	apiInfoURL     string
	fileURL        string
	currentChannel string
	currentVersion string
	httpreq        httpGetter
}

var DefaultChecker = newDefaultChecker()

func newDefaultChecker() *Checker {
	infoURL := constants.APIUpdateInfoURL
	if url, ok := os.LookupEnv("_TEST_UPDATE_INFO_URL"); ok {
		infoURL = url
	}
	updateURL := constants.APIUpdateURL
	if url, ok := os.LookupEnv("_TEST_UPDATE_URL"); ok {
		updateURL = url
	}
	return NewChecker(infoURL, updateURL, constants.BranchName, constants.Version, httpreq.New())
}

func NewChecker(infoURL, fileURL, currentChannel, currentVersion string, httpget httpGetter) *Checker {
	return &Checker{
		infoURL,
		fileURL,
		currentChannel,
		currentVersion,
		httpget,
	}
}

func (u *Checker) Check(cfg Configurable) (*AvailableUpdate, error) {
	return u.CheckFor(cfg, os.Getenv(constants.UpdateBranchEnvVarName), "")
}

func (u *Checker) CheckFor(cfg Configurable, desiredChannel, desiredVersion string) (*AvailableUpdate, error) {
	info, err := u.GetUpdateInfo(cfg, desiredChannel, desiredVersion)
	if err != nil {
		return nil, errs.Wrap(err, "Failed to get update info")
	}

	if info.Channel == u.currentChannel && info.Version == u.currentVersion {
		return nil, nil
	}

	return info, nil
}

func (u *Checker) infoURL(cfg Configurable, desiredVersion, branchName, platform string) string {
	v := make(url.Values)
	v.Set("channel", branchName)
	v.Set("platform", platform)
	v.Set("source", "update")

	if desiredVersion != "" {
		v.Set("target-version", desiredVersion)
	}

	tag := cfg.GetString(CfgUpdateTag)
	if tag != "" {
		v.Set("tag", tag)
	}

	return u.apiInfoURL + "/info?" + v.Encode()
}

func (u *Checker) GetUpdateInfo(cfg Configurable, desiredChannel, desiredVersion string) (*AvailableUpdate, error) {
	if desiredChannel == "" {
		if overrideBranch := os.Getenv(constants.UpdateBranchEnvVarName); overrideBranch != "" {
			desiredChannel = overrideBranch
		} else {
			desiredChannel = u.currentChannel
		}
	}

	infoURL := u.infoURL(cfg, desiredVersion, desiredChannel, runtime.GOOS)
	res, err := u.httpreq.Get(infoURL)
	if err != nil {
		return nil, errs.Wrap(err, "Could not fetch update info from %s", infoURL)
	}

	info := &AvailableUpdate{}
	if err := json.Unmarshal(res, &info); err != nil {
		return nil, errs.Wrap(err, "Could not unmarshal update info: %s", res)
	}

	info.url = u.fileURL + "/" + info.Path

	return info, nil
}

// PrintUpdateMessage will print a message to stdout when an update is available.
// This will only print the message if the current project has a version lock AND if an update is available
func (u *Checker) PrintUpdateMessage(cfg Configurable, pjPath string, out output.Outputer) {
	if versionInfo, _ := projectfile.ParseVersionInfo(pjPath); versionInfo == nil {
		return
	}

	info, err := u.Check(cfg)
	if err != nil {
		logging.Error("Could not check for updates: %v", err)
		return
	}

	if info != nil && info.Version != constants.Version {
		out.Notice(output.Heading(locale.Tl("update_available_title", "Update Available")))
		out.Notice(locale.Tr("update_available", constants.Version, info.Version))
	}
}
