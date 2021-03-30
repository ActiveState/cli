package deprecation

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-version"

	"github.com/ActiveState/cli/internal/constants"
	constvers "github.com/ActiveState/cli/internal/constants/version"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
)

const (
	// DefaultTimeout defines how long we should wait for a response from constants.DeprecationInfoURL
	DefaultTimeout = time.Second

	// fetchKey is the config key used to determine if a deprecation check should occur
	fetchKey = "deprecation_fetch_time"
)

type ErrTimeout struct {
	error
}

// Info details deprecation information for a given version
type Info struct {
	Version     string `json:"version"`
	versionInfo *version.Version
	Date        time.Time `json:"date"`
	DateReached bool
	Reason      string `json:"reason"`
}

// Checker is the struct that we use to do checks with
type Checker struct {
	timeout         time.Duration
	config          Configurable
	deprecationFile string
}

// Configurable defines the configuration function used by the functions in this package
type Configurable interface {
	ConfigPath() string
	GetTime(key string) time.Time
	Set(key string, value interface{}) error
}

// NewChecker returns a new instance of the Checker struct
func NewChecker(timeout time.Duration, configuration Configurable) *Checker {
	return &Checker{
		timeout,
		configuration,
		filepath.Join(configuration.ConfigPath(), "deprecation.json"),
	}
}

// Check will run a Checker.Check with defaults
func Check(cfg Configurable) (*Info, error) {
	return CheckVersionNumber(cfg, constants.VersionNumber)
}

// CheckVersionNumber will run a Checker.Check with defaults
func CheckVersionNumber(cfg Configurable, versionNumber string) (*Info, error) {
	checker := NewChecker(DefaultTimeout, cfg)
	return checker.check(versionNumber)
}

// Check will check if the current version of the tool is deprecated and returns deprecation info if it is.
// This uses a fairly short timeout to check against our deprecation url, so this should not be considered conclusive.
func (checker *Checker) Check() (*Info, error) {
	return checker.check(constants.VersionNumber)
}

func (checker *Checker) check(versionNumber string) (*Info, error) {
	if !constvers.NumberIsProduction(versionNumber) {
		return nil, nil
	}

	versionInfo, err := version.NewVersion(versionNumber)
	if err != nil {
		return nil, errs.Wrap(err, "Invalid version number: %s", versionNumber)
	}

	var infos []Info
	if checker.shouldFetch() {
		infos, err = checker.fetchDeprecationInfo()
		if err != nil {
			return nil, err
		}
	} else {
		infos, err = checker.cachedDeprecationInfo()
		if err != nil {
			return nil, errs.Wrap(err, "cachedDeprecationInfo failed")
		}
	}

	for _, info := range infos {
		if versionInfo.LessThan(info.versionInfo) || versionInfo.Equal(info.versionInfo) {
			return &info, nil
		}
	}

	return nil, nil
}

func (checker *Checker) shouldFetch() bool {
	if fileutils.FileExists(checker.deprecationFile) {
		lastFetch := checker.config.GetTime(fetchKey)
		if !lastFetch.IsZero() && time.Now().Before(lastFetch) {
			return false
		}
	}

	err := checker.config.Set(fetchKey, time.Now().Add(15*time.Minute))
	if err != nil {
		logging.Error("Could not set deprecation fetch time in config, error: %v", err)
	}
	return true

}

func (checker *Checker) fetchDeprecationInfoBody() (int, []byte, error) {
	client := http.Client{
		Timeout: time.Duration(checker.timeout),
	}

	resp, err := client.Get(constants.DeprecationInfoURL)
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

func (checker *Checker) fetchDeprecationInfo() ([]Info, error) {
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

	infos, err := initializeInfo(body)
	if err != nil {
		return nil, errs.Wrap(err, "initializeInfo failed")
	}

	err = checker.saveDeprecationInfo(infos)
	if err != nil {
		return nil, errs.Wrap(err, "saveDeprecatinInfo failed")
	}

	return infos, nil
}

func (checker *Checker) saveDeprecationInfo(info []Info) error {
	data, err := json.MarshalIndent(info, "", " ")
	if err != nil {
		return locale.WrapError(err, "err_save_deprection", "Could not save deprication information")
	}

	return ioutil.WriteFile(checker.deprecationFile, data, 0644)
}

func (checker *Checker) cachedDeprecationInfo() ([]Info, error) {
	logging.Debug("Using cached deprecation information")
	data, err := ioutil.ReadFile(checker.deprecationFile)
	if err != nil {
		return nil, locale.WrapError(err, "err_read_deprection", "Could not read cached deprecation information")
	}

	info, err := initializeInfo(data)
	if err != nil {
		return nil, locale.WrapError(err, "err_init_deprecation_info", "Could not initialize deprecation information")
	}

	return info, nil
}

func initializeInfo(data []byte) ([]Info, error) {
	var info []Info
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
