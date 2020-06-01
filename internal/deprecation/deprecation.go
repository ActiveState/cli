package deprecation

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/spf13/viper"

	"github.com/ActiveState/cli/internal/constants"
	constvers "github.com/ActiveState/cli/internal/constants/version"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
)

const (
	// DefaultTimeout defines how long we should wait for a response from constants.DeprecationInfoURL
	DefaultTimeout = time.Second

	// timeKey is the config key used to determine if a deprecation check should occur
	timeKey = "deprecation_time"
)

var (
	// FailFetchDeprecationInfo communicates a failure in retrieving the deprecation info via http
	FailFetchDeprecationInfo = failures.Type("deprecation.fail.info", failures.FailNetwork)

	// FailParseVersion communicates a failure in parsing a semantic version (the version is not formatted properly)
	FailParseVersion = failures.Type("deprecation.fail.versionparse", failures.FailInput)

	// FailTimeout communicates a failure due to a timeout
	FailTimeout = failures.Type("deprecation.fail.timeout", failures.FailNetwork, failures.FailNonFatal)

	// FailNotFound communicates a failure due to a 404
	FailNotFound = failures.Type("deprecation.fail.notfound", failures.FailNotFound, failures.FailNetwork, failures.FailNonFatal)

	// FailInvalidResponseCode communicates a failure due to a non-200 response code
	FailInvalidResponseCode = failures.Type("deprecation.fail.code", failures.FailNetwork)
)

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
	timeout time.Duration
	config  configable
}

type configable interface {
	GetTime(key string) time.Time
	Set(key string, value interface{})
}

// NewChecker returns a new instance of the Checker struct
func NewChecker(timeout time.Duration, config configable) *Checker {
	return &Checker{timeout, config}
}

// Check will run a Checker.Check with defaults
func Check() (*Info, *failures.Failure) {
	return CheckVersionNumber(constants.VersionNumber)
}

// CheckVersionNumber will run a Checker.Check with defaults
func CheckVersionNumber(versionNumber string) (*Info, *failures.Failure) {
	checker := NewChecker(DefaultTimeout, viper.GetViper())
	return checker.check(versionNumber)
}

// Check will check if the current version of the tool is deprecated and returns deprecation info if it is.
// This uses a fairly short timeout to check against our deprecation url, so this should not be considered conclusive.
func (checker *Checker) Check() (*Info, *failures.Failure) {
	return checker.check(constants.VersionNumber)
}

func (checker *Checker) check(versionNumber string) (*Info, *failures.Failure) {
	if !checker.shouldCheck(versionNumber) {
		return nil, nil
	}

	versionInfo, err := version.NewVersion(versionNumber)
	if err != nil {
		return nil, FailParseVersion.Wrap(err)
	}

	infos, fail := checker.fetchDeprecationInfo()
	if fail != nil {
		return nil, fail
	}

	for _, info := range infos {
		if versionInfo.LessThan(info.versionInfo) || versionInfo.Equal(info.versionInfo) {
			return &info, nil
		}
	}

	return nil, nil
}

func (checker *Checker) shouldCheck(versionNumber string) bool {
	if !constvers.NumberIsProduction(versionNumber) {
		return false
	}

	lastCheck := checker.config.GetTime(timeKey)
	if !lastCheck.IsZero() && time.Now().Before(lastCheck) {
		return false
	}

	checker.config.Set(timeKey, time.Now().Add(15*time.Minute))
	return true

}

func (checker *Checker) fetchDeprecationInfoBody() (int, []byte, *failures.Failure) {
	client := http.Client{
		Timeout: time.Duration(checker.timeout),
	}

	resp, err := client.Get(constants.DeprecationInfoURL)
	if err != nil {
		// Check for timeout by evaluating the error string. Yeah this is dumb, thank the http package for that.
		if strings.Contains(err.Error(), "Client.Timeout") || strings.Contains(err.Error(), "context deadline exceeded") {
			return -1, nil, FailTimeout.Wrap(err)
		}
		return -1, nil, FailFetchDeprecationInfo.Wrap(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return -1, nil, failures.FailIO.Wrap(err)
	}

	return resp.StatusCode, body, nil
}

func (checker *Checker) fetchDeprecationInfo() ([]Info, *failures.Failure) {
	code, body, fail := checker.fetchDeprecationInfoBody()
	if fail != nil {
		if fail.Type.Matches(FailTimeout) {
			logging.Debug("Timed out while fetching deprecation info: %v", fail.Error())
			return nil, nil
		}
		return nil, fail
	}

	// Handle non-200 response gracefully
	if code != 200 {
		if code == 404 || code == 403 { // On S3 a 403 means a 404, at least for our use-case
			return nil, FailNotFound.New(locale.T("err_deprection_404"))
		}
		return nil, FailInvalidResponseCode.New(locale.Tr("err_deprection_code", strconv.Itoa(code)))
	}

	infos := make([]Info, 0)
	err := json.Unmarshal(body, &infos)
	if err != nil {
		return nil, failures.FailMarshal.Wrap(err)
	}

	for k := range infos {
		infos[k].versionInfo, err = version.NewVersion(infos[k].Version)
		if err != nil {
			return nil, FailParseVersion.Wrap(err)
		}
		infos[k].DateReached = infos[k].Date.Before(time.Now())
	}

	sort.Slice(infos, func(i, j int) bool {
		return infos[i].versionInfo.GreaterThan(infos[j].versionInfo)
	})

	return infos, nil
}
