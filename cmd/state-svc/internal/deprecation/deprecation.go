package deprecation

import (
	"encoding/json"
	"io/ioutil"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/graph"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/retryhttp"
	"github.com/hashicorp/go-version"
	"github.com/patrickmn/go-cache"
)

const (
	cacheKey = "info"
)

type ErrTimeout struct {
	error
}

// info details deprecation information for a given version
type info struct {
	Version     string `json:"version"`
	versionInfo *version.Version
	Date        time.Time `json:"date"`
	DateReached bool
	Reason      string `json:"reason"`
}

// Checker is the struct that we use to do checks with
type Checker struct {
	config configurable
	cache  *cache.Cache
	done   chan struct{}
}

// configurable defines the configuration function used by the functions in this package
type configurable interface {
	ConfigPath() string
	GetTime(key string) time.Time
	Set(key string, value interface{}) error
	Close() error
}

// NewChecker returns a new instance of the Checker struct
func NewChecker(configuration configurable) *Checker {
	checker := &Checker{
		configuration,
		cache.New(15*time.Minute, 5*time.Minute),
		make(chan struct{}),
	}

	go checker.pollDeprecationInfo()

	return checker
}

// Check will Check if the current version of the tool is deprecated and returns deprecation info if it is.
// This uses a fairly short timeout to Check against our deprecation url, so this should not be considered conclusive.
func (checker *Checker) Check() (*graph.DeprecationInfo, error) {
	data, exists := checker.cache.Get(cacheKey)
	if !exists {
		return nil, nil
	}

	infos, ok := data.([]info)
	if !ok {
		return nil, errs.New("Unexpected cache entry for deprecation info")
	}

	versionInfo, err := version.NewVersion(constants.Version)
	if err != nil {
		return nil, errs.Wrap(err, "Invalid version number: %s", constants.Version)
	}

	for _, info := range infos {
		if versionInfo.LessThan(info.versionInfo) || versionInfo.Equal(info.versionInfo) {
			return &graph.DeprecationInfo{
				Version:     info.Version,
				Date:        info.Date.Format(constants.DateFormatUser),
				DateReached: info.DateReached,
				Reason:      info.Reason,
			}, nil
		}
	}

	return nil, nil
}

func (checker *Checker) pollDeprecationInfo() {
	timer := time.NewTicker(1 * time.Hour)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			err := checker.Refresh()
			if err != nil {
				multilog.Critical("Could not update deprecation information %s", errs.JoinMessage(err))
				return
			}
		case <-checker.done:
			return
		}
	}
}

func (checker *Checker) Refresh() error {
	deprecated, err := checker.fetchDeprecationInfo()
	if err != nil {
		return errs.Wrap(err, "Could not fetch deprecation information")
	}
	if deprecated != nil {
		checker.cache.Set(cacheKey, deprecated, cache.DefaultExpiration)
	}
	return nil
}

func (checker *Checker) fetchDeprecationInfo() ([]info, error) {
	logging.Debug("Fetching deprecation information from S3")

	code, body, err := checker.fetchDeprecationInfoBody()
	if err != nil {
		if errs.Matches(err, &ErrTimeout{}) {
			logging.Debug("Timed out while fetching deprecation info: %v", err)
			return nil, nil
		}
		return nil, err
	}

	// Handle non-200 response gracefully
	if code != 200 {
		if code == 404 || code == 403 { // On S3 a 403 means a 404, at least for our use-case
			return nil, locale.NewError("err_deprection_404")
		}
		return nil, locale.NewError("err_deprection_code", "", strconv.Itoa(code))
	}

	return initializeInfo(body)
}

func (checker *Checker) fetchDeprecationInfoBody() (int, []byte, error) {
	resp, err := retryhttp.DefaultClient.Get(constants.DeprecationInfoURL)
	if err != nil {
		// Check for timeout by evaluating the error string. Yeah this is dumb, thank the http package for that.
		if strings.Contains(err.Error(), "Client.Timeout") || strings.Contains(err.Error(), "context deadline exceeded") {
			return -1, nil, &ErrTimeout{errs.Wrap(err, "timed out")}
		}
		return -1, nil, errs.Wrap(err, "Could not fetch deprecation info")
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return -1, nil, errs.Wrap(err, "Read body failed")
	}

	return resp.StatusCode, body, nil
}

func initializeInfo(data []byte) ([]info, error) {
	var info []info
	err := json.Unmarshal(data, &info)
	if err != nil {
		return nil, locale.WrapError(err, "err_unmarshal_deprecation", "Could not unmarshall deprecation information")
	}

	for k := range info {
		info[k].versionInfo, err = version.NewVersion(info[k].Version)
		if err != nil {
			return nil, locale.WrapError(err, "err_deprecation_parse_version", "Could not parse version in deprecation information")
		}
		info[k].DateReached = info[k].Date.Before(time.Now())
	}

	sort.Slice(info, func(i, j int) bool {
		return info[i].versionInfo.GreaterThan(info[j].versionInfo)
	})

	return info, nil
}

func (checker *Checker) Close() {
	logging.Debug("Closing deprecation checker")
	checker.done <- struct{}{}
}
