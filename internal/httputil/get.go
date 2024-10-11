package httputil

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/proxyreader"
	"github.com/ActiveState/cli/internal/retryhttp"
	"github.com/ActiveState/cli/pkg/runtime/events/progress"
)

// Get takes a URL and returns the contents as bytes
var Get func(url string) ([]byte, error)

var GetDirect = httpGet

// GetWithProgress takes a URL and returns the contents as bytes, it takes an optional second arg which will spawn a progressbar
var GetWithProgress func(url string, progress progress.Reporter) ([]byte, error)

func init() {
	SetMocking(condition.InUnitTest())
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
	return httpGetWithProgress(url, nil)
}

func httpGetWithProgress(url string, progress progress.Reporter) ([]byte, error) {
	return httpGetWithProgressRetry(url, progress, 1, 3)
}

func httpGetWithProgressRetry(url string, prg progress.Reporter, attempt int, retries int) ([]byte, error) {
	client := retryhttp.NewClient(0 /* 0 = no timeout */, retries)
	resp, err := client.Get(url)
	if err != nil {
		code := -1
		if resp != nil {
			code = resp.StatusCode
		}
		return nil, locale.WrapError(err, "err_network_get", "", "Status code: {{.V0}}", strconv.Itoa(code))
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, locale.NewError("err_invalid_status_code", "", strconv.Itoa(resp.StatusCode))
	}

	var total int
	length := resp.Header.Get("Content-Length")
	if length == "" {
		total = 1
	} else {
		total, err = strconv.Atoi(length)
		if err != nil {
			return nil, errs.Wrap(err, "Could not convert header length to int, value: %s", length)
		}
	}

	var src io.Reader = resp.Body
	defer resp.Body.Close()

	if prg != nil && attempt == 1 {
		if err := prg.ReportSize(total); err != nil {
			return nil, errs.Wrap(err, "Could not report size")
		}
		src = proxyreader.NewProxyReader(prg, resp.Body)
	}

	var dst bytes.Buffer
	_, err = io.Copy(&dst, src)
	if err != nil && !errors.Is(err, io.EOF) {
		logging.Debug("Reading body failed: %s", errs.JoinMessage(err))
		if attempt <= retries {
			return httpGetWithProgressRetry(url, prg, attempt+1, retries)
		}
		return nil, errs.Wrap(err, "Could not copy network stream")
	}

	return dst.Bytes(), nil
}

func _testHTTPGetWithProgress(url string, progress progress.Reporter) ([]byte, error) {
	return _testHTTPGet(url)
}

// _testHTTPGet is used when in tests, this cannot be in the test itself as that would limit it to only that one test
func _testHTTPGet(url string) ([]byte, error) {
	path := strings.Replace(url, constants.APIArtifactURL, "", 1)
	path = filepath.Join(environment.GetRootPathUnsafe(), "test", path)

	body, err := os.ReadFile(path)
	if err != nil {
		return nil, errs.Wrap(err, "Could not read file contents: %s", path)
	}

	return body, nil
}
