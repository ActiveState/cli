package download

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
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
	"github.com/hashicorp/go-retryablehttp"
)

// Get takes a URL and returns the contents as bytes
var Get func(req *Request) ([]byte, error)

var GetURL func(url string) ([]byte, error)

var GetDirect = httpGet

type DownloadProgress interface {
	TotalSize(int)
	IncrBy(int)
}

// GetWithProgress takes a URL and returns the contents as bytes, it takes an optional second arg which will spawn a progressbar
var GetWithProgress func(req *Request, progress DownloadProgress) ([]byte, error)

type Request struct {
	*retryablehttp.Request
}

func init() {
	SetMocking(condition.InUnitTest())
}

// SetMocking sets the correct Get methods for testing
func SetMocking(useMocking bool) {
	if useMocking {
		Get = _testHTTPGet
		GetURL = _testHTTPGetURL
		GetWithProgress = _testHTTPGetWithProgress
	} else {
		Get = httpGet
		GetURL = httpGetURL
		GetWithProgress = httpGetWithProgress
	}
}

func NewRequest(url string) (*Request, error) {
	req, err := retryablehttp.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, errs.Wrap(err, "Could not intialize new retryable http request")
	}

	return &Request{req}, nil
}

func httpGetURL(url string) ([]byte, error) {
	req, err := NewRequest(url)
	if err != nil {
		return nil, errs.Wrap(err, "Could not create new request")
	}

	return httpGet(req)
}

func httpGet(req *Request) ([]byte, error) {
	logging.Debug("Retrieving url: %s", req.URL.String())
	return httpGetWithProgress(req, nil)
}

func httpGetWithProgress(req *Request, progress DownloadProgress) ([]byte, error) {
	return httpGetWithProgressRetry(req, progress, 1, 3)
}

func httpGetWithProgressRetry(req *Request, prg DownloadProgress, attempt int, retries int) ([]byte, error) {
	logging.Debug("Retrieving url: %s, attempt: %d", req.URL.String(), attempt)
	client := retryhttp.NewClient(0 /* 0 = no timeout */, retries)
	resp, err := client.Do(req.Request)
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
			logging.Debug("Content-length: %v", length)
			return nil, errs.Wrap(err, "Could not convert header length to int, value: %s", length)
		}
	}

	var src io.Reader = resp.Body
	defer resp.Body.Close()

	if prg != nil {
		prg.TotalSize(total)
		src = proxyreader.NewProxyReader(prg, resp.Body)
	}

	var dst bytes.Buffer
	_, err = io.Copy(&dst, src)
	if err != nil {
		logging.Debug("Reading body failed: %s", err)
		if attempt <= retries {
			return httpGetWithProgressRetry(req, prg, attempt+1, retries)
		}
		return nil, errs.Wrap(err, "Could not copy network stream")
	}

	return dst.Bytes(), nil
}

func _testHTTPGetURL(url string) ([]byte, error) {
	req, err := NewRequest(url)
	if err != nil {
		return nil, errs.Wrap(err, "Could not create new request")
	}

	return _testHTTPGet(req)
}

func _testHTTPGetWithProgress(req *Request, progress DownloadProgress) ([]byte, error) {
	return _testHTTPGet(req)
}

// _testHTTPGet is used when in tests, this cannot be in the test itself as that would limit it to only that one test
func _testHTTPGet(req *Request) ([]byte, error) {
	path := strings.Replace(req.URL.String(), constants.APIArtifactURL, "", 1)
	path, err := url.QueryUnescape(path)
	if err != nil {
		return nil, errs.Wrap(err, "Could not unescape path: %s", path)
	}
	path = filepath.Join(environment.GetRootPathUnsafe(), "test", path)

	body, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errs.Wrap(err, "Could not read file contents: %s", path)
	}

	return body, nil
}
