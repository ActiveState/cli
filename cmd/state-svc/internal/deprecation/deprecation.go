package deprecation

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/graph"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/retryhttp"
	"github.com/hashicorp/go-version"
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
	}

	return checker
}

// Check will Check if the current version of the tool is deprecated and returns deprecation info if it is.
// This uses a fairly short timeout to Check against our deprecation url, so this should not be considered conclusive.
func (checker *Checker) Check() (*graph.DeprecationInfo, error) {
	deprecationInfo, err := checker.fetchDeprecationInfo()
	if err != nil {
		return nil, errs.Wrap(err, "Could not fetch deprecation information")
	}

	return deprecationInfo, nil
}

func (checker *Checker) fetchDeprecationInfo() (*graph.DeprecationInfo, error) {
	logging.Debug("Fetching deprecation information")

	body, err := checker.fetchDeprecationInfoBody()
	if err != nil {
		if errs.Matches(err, &ErrTimeout{}) {
			logging.Debug("Timed out while fetching deprecation info: %v", err)
			return nil, nil
		}
		return nil, err
	}

	logging.Debug("Received: %s", string(body))

	infos, err := initializeInfo(body)
	if err != nil {
		return nil, errs.Wrap(err, "Could not intialize deprecation info")
	}

	versionInfo, err := version.NewVersion(constants.VersionNumber)
	if err != nil {
		return nil, errs.Wrap(err, "Invalid version number: %s", constants.Version)
	}

	for _, info := range infos {
		logging.Debug("Comparing %s to %s", versionInfo.String(), info.versionInfo.String())
		if versionInfo.LessThan(info.versionInfo) || versionInfo.Equal(info.versionInfo) {
			logging.Debug("Found version to be deprecated")
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

func (checker *Checker) fetchDeprecationInfoBody() ([]byte, error) {
	if f := os.Getenv(constants.DeprecationOverrideEnvVarName); f != "" {
		v, err := fileutils.ReadFile(f)
		if err != nil {
			return nil, errs.Wrap(err, "Could not read provided deprecation info file: %s", f)
		}
		return v, nil
	}

	logging.Debug("Fetching deprecation information from S3")

	resp, err := retryhttp.DefaultClient.Get(constants.DeprecationInfoURL)
	if err != nil {
		// Check for timeout by evaluating the error string. Yeah this is dumb, thank the http package for that.
		if strings.Contains(err.Error(), "Client.Timeout") || strings.Contains(err.Error(), "context deadline exceeded") {
			return nil, &ErrTimeout{errs.Wrap(err, "timed out")}
		}
		return nil, errs.Wrap(err, "Could not fetch deprecation info")
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errs.Wrap(err, "Read body failed")
	}

	// Handle non-200 response gracefully
	if resp.StatusCode != 200 {
		if resp.StatusCode == 404 || resp.StatusCode == 403 { // On S3 a 403 means a 404, at least for our use-case
			return nil, locale.NewError("err_deprection_404")
		}
		return nil, locale.NewError("err_deprection_code", "", strconv.Itoa(resp.StatusCode))
	}

	return body, nil
}

func initializeInfo(data []byte) ([]info, error) {
	var info []info
	err := json.Unmarshal(data, &info)
	if err != nil {
		return nil, locale.WrapError(err, "err_unmarshal_deprecation", "Could not unmarshall deprecation information: %s", string(data))
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

