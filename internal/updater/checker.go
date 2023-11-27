package updater

import (
	"encoding/json"
	"net/url"
	"os"
	"runtime"
	"strings"

	"github.com/ActiveState/cli/internal/analytics"
	anaConst "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/analytics/dimensions"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/httpreq"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
)

type httpGetter interface {
	Get(string) ([]byte, int, error)
}

type Configurable interface {
	GetString(string) string
	Set(string, interface{}) error
}

type InvocationSource string

var (
	InvocationSourceInstall InvocationSource = "install"
	InvocationSourceUpdate  InvocationSource = "update"
)

type Checker struct {
	cfg        Configurable
	an         analytics.Dispatcher
	apiInfoURL string
	httpreq    httpGetter
	cache      *AvailableUpdate
	done       chan struct{}

	InvocationSource InvocationSource
}

func NewDefaultChecker(cfg Configurable, an analytics.Dispatcher) *Checker {
	infoURL := constants.APIUpdateInfoURL
	if url, ok := os.LookupEnv("_TEST_UPDATE_INFO_URL"); ok {
		infoURL = url
	}
	return NewChecker(cfg, an, infoURL, httpreq.New())
}

func NewChecker(cfg Configurable, an analytics.Dispatcher, infoURL string, httpget httpGetter) *Checker {
	return &Checker{
		cfg,
		an,
		infoURL,
		httpget,
		nil,
		make(chan struct{}),
		InvocationSourceUpdate,
	}
}

func (u *Checker) CheckFor(desiredChannel, desiredVersion string) (*AvailableUpdate, error) {
	info, err := u.getUpdateInfo(desiredChannel, desiredVersion)
	if err != nil {
		return nil, errs.Wrap(err, "Failed to get update info")
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

func (u *Checker) getUpdateInfo(desiredChannel, desiredVersion string) (*AvailableUpdate, error) {
	tag := u.cfg.GetString(CfgUpdateTag)
	infoURL := u.infoURL(tag, desiredVersion, desiredChannel, runtime.GOOS)
	logging.Debug("Getting update info: %s", infoURL)
	var label string
	var msg string
	res, code, err := u.httpreq.Get(infoURL)
	if err != nil {
		if code == 404 || strings.Contains(string(res), "Could not retrieve update info") {
			// The above string match can be removed once https://www.pivotaltracker.com/story/show/179426519 is resolved
			logging.Debug("Update info 404s: %v", errs.JoinMessage(err))
			label = anaConst.UpdateLabelUnavailable
			msg = anaConst.UpdateErrorNotFound
			err = nil
		} else if code == 403 || code == 503 {
			// The request could not be satisfied or service is unavailable. This happens when Cloudflare
			// blocks access, or the service is unavailable in a particular geographic location.
			logging.Warning("Update info request blocked or service unavailable: %v", err)
			label = anaConst.UpdateLabelUnavailable
			msg = anaConst.UpdateErrorBlocked
			err = nil
		} else {
			label = anaConst.UpdateLabelFailed
			msg = anaConst.UpdateErrorFetch
			err = errs.Wrap(err, "Could not fetch update info from %s", infoURL)
		}

		u.an.EventWithLabel(
			anaConst.CatUpdates,
			anaConst.ActUpdateCheck,
			label,
			&dimensions.Values{
				Version: ptr.To(desiredVersion),
				Error:   ptr.To(msg),
			},
		)
		return nil, err
	}

	var info *AvailableUpdate
	if err := json.Unmarshal(res, &info); err != nil {
		return nil, errs.Wrap(err, "Could not unmarshal update info: %s", res)
	}

	u.an.EventWithLabel(
		anaConst.CatUpdates,
		anaConst.ActUpdateCheck,
		anaConst.UpdateLabelAvailable,
		&dimensions.Values{
			Version: ptr.To(info.Version),
		},
	)

	return info, nil
}
