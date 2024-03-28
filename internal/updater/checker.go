package updater

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"

	"github.com/ActiveState/cli/internal/analytics"
	anaConst "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/analytics/dimensions"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/retryhttp"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
)

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
	retryhttp  *retryhttp.Client
	cache      *AvailableUpdate
	done       chan struct{}

	InvocationSource InvocationSource
}

func NewDefaultChecker(cfg Configurable, an analytics.Dispatcher) *Checker {
	infoURL := constants.APIUpdateInfoURL
	if url, ok := os.LookupEnv("_TEST_UPDATE_INFO_URL"); ok {
		infoURL = url
	}
	return NewChecker(cfg, an, infoURL, retryhttp.DefaultClient)
}

func NewChecker(cfg Configurable, an analytics.Dispatcher, infoURL string, httpget *retryhttp.Client) *Checker {
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

	var info *AvailableUpdate
	var err error
	var label string
	var msg string
	dims := &dimensions.Values{Version: ptr.To(desiredVersion)} // will change to info.Version if possible

	var resp *http.Response
	if resp, err = u.retryhttp.Get(infoURL); err == nil {
		var res []byte
		res, err = io.ReadAll(resp.Body)
		switch {
		// If there was an error reading the response.
		case err != nil:
			label = anaConst.UpdateLabelFailed
			msg = anaConst.UpdateErrorFetch
			err = errs.Wrap(err, "Could not read update info")

		// If the response was a 404 not found, or if the response body indicates failure.
		// The above string match can be removed once https://www.pivotaltracker.com/story/show/179426519 is resolved
		case resp.StatusCode == 404 || strings.Contains(string(res), "Could not retrieve update info"):
			logging.Debug("Update info 404s: %v", errs.JoinMessage(err))
			label = anaConst.UpdateLabelUnavailable
			msg = anaConst.UpdateErrorNotFound

		// The request could not be satisfied or service is unavailable. This happens when Cloudflare
		// blocks access, or the service is unavailable in a particular geographic location.
		case resp.StatusCode == 403 || resp.StatusCode == 503:
			logging.Warning("Update info request blocked or service unavailable: %v", err)
			label = anaConst.UpdateLabelUnavailable
			msg = anaConst.UpdateErrorBlocked

		// If all went well.
		default:
			if err = json.Unmarshal(res, &info); err == nil {
				label = anaConst.UpdateLabelAvailable
				dims.Version = ptr.To(info.Version)
			} else {
				label = anaConst.UpdateLabelFailed
				msg = anaConst.UpdateErrorFetch
				err = errs.Wrap(err, "Could not unmarshal update info: %s", res)
			}
		}

	} else { // retryhttp returned err
		label = anaConst.UpdateLabelFailed
		msg = anaConst.UpdateErrorFetch
		err = errs.Wrap(err, "Could not fetch update info from %s", infoURL)
		if e, ok := err.(net.Error); ok && e.Timeout() {
			logging.Debug("Silencing network timeout error: %v", err)
			err = errs.Silence(err)
		}
	}

	if msg != "" {
		dims.Error = ptr.To(msg)
	}

	u.an.EventWithLabel(anaConst.CatUpdates, anaConst.ActUpdateCheck, label, dims)

	return info, err
}
