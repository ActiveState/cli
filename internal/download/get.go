package download

import (
	"bytes"
	"io"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/progress"
	"github.com/ActiveState/cli/internal/retryfn"
	"github.com/ActiveState/cli/internal/retryhttp"
)

// Get takes a URL and returns the contents as bytes
var Get func(url string) ([]byte, error)

// GetWithProgress takes a URL and returns the contents as bytes, it takes an optional second arg which will spawn a progressbar
var GetWithProgress func(url string, progress *progress.Progress) ([]byte, error)

func init() {
	SetMocking(condition.InTest())
}

// SetMocking sets the correct Get methods for testing
func SetMocking(useMocking bool) {
	if useMocking {
		Get = _testHTTPGet
		GetWithProgress = _testHTTPGetWithProgress
	} else {
		Get = httpGet
		GetWithProgress = httpGetWithProgress
	}
}

func httpGet(url string) ([]byte, error) {
	logging.Debug("Retrieving url: %s", url)
	return httpGetWithProgress(url, nil)
}

func httpGetWithProgress(url string, progress *progress.Progress) ([]byte, error) {
	var bs []byte
	fn := func() error {
		logging.Debug("Retrieving url: %s", url)
		client := retryhttp.NewClientFromExisting(retryhttp.DefaultClient, 3)
		resp, err := client.Get(url)
		if err != nil {
			code := -1
			if resp != nil {
				code = resp.StatusCode
			}
			fail := failures.FailNetwork.Wrap(err, locale.Tl("err_network_get", "Status code: {{.V0}}", strconv.Itoa(code)))
			return &retryfn.ControlError{
				Cause: fail,
				Type:  retryfn.Halt,
			}
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			fail := failures.FailNetwork.New("err_invalid_status_code", strconv.Itoa(resp.StatusCode))
			return &retryfn.ControlError{
				Cause: fail,
				Type:  retryfn.Halt,
			}
		}

		var total int
		length := resp.Header.Get("Content-Length")
		if length == "" {
			total = 1
		} else {
			total, err = strconv.Atoi(length)
			if err != nil {
				logging.Debug("Content-length: %v", length)
				return failures.FailInput.Wrap(err)
			}
		}

		bar := progress.AddByteProgressBar(int64(total))
		src := bar.ProxyReader(resp.Body)
		var dst bytes.Buffer

		_, err = io.Copy(&dst, src)
		if err != nil {
			return failures.FailInput.Wrap(err)
		}

		if !bar.Completed() {
			// Failsafe, so we don't get blocked by a progressbar
			bar.IncrBy(total)
		}

		bs = dst.Bytes()

		return nil

	}

	retryFn := retryfn.New(3, fn)
	return bs, retryFn.Run()
}

func _testHTTPGetWithProgress(url string, progress *progress.Progress) ([]byte, error) {
	return _testHTTPGet(url)
}

// _testHTTPGet is used when in tests, this cannot be in the test itself as that would limit it to only that one test
func _testHTTPGet(url string) ([]byte, error) {
	path := strings.Replace(url, constants.APIArtifactURL, "", 1)
	path = filepath.Join(environment.GetRootPathUnsafe(), "test", path)

	body, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, failures.FailIO.Wrap(err)
	}

	return body, nil
}
