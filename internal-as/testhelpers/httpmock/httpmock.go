package httpmock

import (
	"fmt"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"

	parent "github.com/jarcoal/httpmock"

	"github.com/ActiveState/cli/internal-as/fileutils"
	"github.com/ActiveState/cli/internal-as/logging"
)

// HTTPMock encapsulate the functionality for mocking requests to a specific base-url.
type HTTPMock struct {
	urlPrefix string
}

var httpMocks = map[string]*HTTPMock{}
var defaultMock *HTTPMock

// Activate an and return an *HTTPMock instance. If none yet activated, then the first becomes the default
// HTTPMock; which means you can just call package funcs to use it.
func Activate(prefix string) *HTTPMock {
	urlPrefix := strings.TrimSuffix(prefix, "/")
	if _, mockExists := httpMocks[urlPrefix]; mockExists {
		logging.Warning("Activating http mock for a prefix that is already in use, this could cause issues. Prefix=%s", urlPrefix)
	}

	mock := &HTTPMock{urlPrefix: urlPrefix}
	httpMocks[urlPrefix] = mock

	if defaultMock == nil {
		defaultMock = mock
		parent.Activate()
	}

	return mock
}

// DeActivate the httpmock
func DeActivate() {
	defer parent.DeactivateAndReset()
	defaultMock = nil
	httpMocks = map[string]*HTTPMock{}
}

// Register registers a httpmock for the given request (the response file is based on the request)
func (mock *HTTPMock) Register(method string, request string) {
	mock.RegisterWithCode(method, request, 200)
}

// RegisterWithCode is the same as Register but it allows specifying a code
func (mock *HTTPMock) RegisterWithCode(method string, request string, code int) {
	responseFile := strings.Replace(request, mock.urlPrefix, "", 1)
	mock.RegisterWithResponse(method, request, code, responseFile)
}

// RegisterWithResponse is the same as RegisterWithCode but it allows specifying a response file
func (mock *HTTPMock) RegisterWithResponse(method string, request string, code int, responseFile string) {
	responsePath := getResponsePath()
	responseFile = getResponseFile(method, code, responseFile, responsePath)
	request = strings.TrimSuffix(mock.urlPrefix+"/"+strings.TrimPrefix(request, "/"), "/")
	parent.RegisterResponder(method, request,
		parent.NewStringResponder(code, string(fileutils.ReadFileUnsafe(responseFile))))
}

// RegisterWithResponseBody will respond with the given code and responseBody, no external files involved
func (mock *HTTPMock) RegisterWithResponseBody(method string, request string, code int, responseBody string) {
	mock.RegisterWithResponseBytes(method, request, code, []byte(responseBody))
}

// RegisterWithResponseBytes will respond with the given code and responseBytes, no external files involved
func (mock *HTTPMock) RegisterWithResponseBytes(method string, request string, code int, responseBytes []byte) {
	request = strings.TrimSuffix(mock.urlPrefix+"/"+strings.TrimPrefix(request, "/"), "/")
	parent.RegisterResponder(method, request,
		parent.NewBytesResponder(code, responseBytes))
}

// RegisterWithResponderBody register a httpmock with a custom responder that returns the response body
func (mock *HTTPMock) RegisterWithResponderBody(method string, request string, cb func(req *http.Request) (int, string)) {
	request = strings.TrimSuffix(mock.urlPrefix+"/"+strings.TrimPrefix(request, "/"), "/")
	parent.RegisterResponder(method, request, func(req *http.Request) (*http.Response, error) {
		code, responseData := cb(req)
		return parent.NewStringResponse(code, responseData), nil
	})
}

// RegisterWithResponder register a httpmock with a custom responder
func (mock *HTTPMock) RegisterWithResponder(method string, request string, cb func(req *http.Request) (int, string)) {
	request = strings.TrimSuffix(mock.urlPrefix+"/"+strings.TrimPrefix(request, "/"), "/")
	logging.Debug("Mocking: %s", request)
	responsePath := getResponsePath()
	parent.RegisterResponder(method, request, func(req *http.Request) (*http.Response, error) {
		code, responseFile := cb(req)
		responseFile = getResponseFile(method, code, responseFile, responsePath)
		responseData := string(fileutils.ReadFileUnsafe(responseFile))
		return parent.NewStringResponse(code, responseData), nil
	})
}

func getResponseFile(method string, code int, responseFile string, responsePath string) string {
	responseFile = fmt.Sprintf("%s-%s", strings.ToUpper(method), strings.TrimPrefix(responseFile, "/"))
	if code != 200 {
		responseFile = fmt.Sprintf("%s-%d", responseFile, code)
	}
	ext := ".json"
	if filepath.Ext(responseFile) != "" {
		ext = ""
	}
	responseFile = filepath.Join(responsePath, responseFile) + ext

	return responseFile
}

func getResponsePath() string {
	_, currentFile, _, _ := runtime.Caller(0)
	file := currentFile
	ok := true
	iter := 2

	for file == currentFile && ok {
		_, file, _, ok = runtime.Caller(iter)
		iter = iter + 1
	}

	if file == "" || file == currentFile {
		panic("Could not get caller")
	}

	return filepath.Join(filepath.Dir(file), "testdata", "httpresponse")
}

func ensureDefaultMock() {
	if defaultMock == nil {
		panic("default HTTPMock is not defined")
	}
}

// Register defers to the default HTTPMock's Register or errors if no default defined.
func Register(method string, request string) {
	ensureDefaultMock()
	defaultMock.Register(method, request)
}

// RegisterWithCode defers to the default HTTPMock's RegisterWithCode or errors if no default defined.
func RegisterWithCode(method string, request string, code int) {
	ensureDefaultMock()
	defaultMock.RegisterWithCode(method, request, code)
}

// RegisterWithResponse defers to the default HTTPMock's RegisterWithResponse or errors if no default defined.
func RegisterWithResponse(method string, request string, code int, responseFile string) {
	ensureDefaultMock()
	defaultMock.RegisterWithResponse(method, request, code, responseFile)
}

// RegisterWithResponseBody defers to the default HTTPMock's RegisterWithResponseBody or errors if no default defined.
func RegisterWithResponseBody(method string, request string, code int, responseBody string) {
	ensureDefaultMock()
	defaultMock.RegisterWithResponseBody(method, request, code, responseBody)
}

// RegisterWithResponseBytes defers to the default HTTPMock's RegisterWithResponseBytes or errors if no default defined.
func RegisterWithResponseBytes(method string, request string, code int, responseBytes []byte) {
	ensureDefaultMock()
	defaultMock.RegisterWithResponseBytes(method, request, code, responseBytes)
}

// RegisterWithResponderBody defers to the default HTTPMock's RegisterWithResponderBody or errors if no default defined.
func RegisterWithResponderBody(method string, request string, code int, cb func(req *http.Request) (int, string)) {
	ensureDefaultMock()
	defaultMock.RegisterWithResponderBody(method, request, cb)
}

// RegisterWithResponder defers to the default HTTPMock's RegisterWithResponder or errors if no default defined.
func RegisterWithResponder(method string, request string, cb func(req *http.Request) (int, string)) {
	ensureDefaultMock()
	defaultMock.RegisterWithResponder(method, request, cb)
}
