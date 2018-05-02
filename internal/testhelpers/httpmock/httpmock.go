package httpmock

import (
	"fmt"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/print"
	parent "gopkg.in/jarcoal/httpmock.v1"
)

var urlPrefix string

// Activate the httpmock
func Activate(prefix string) {
	if urlPrefix != "" {
		panic("You already have an active httpmock, deactivate the old one first")
	}
	urlPrefix = strings.TrimSuffix(prefix, "/")
	parent.Activate()
}

// DeActivate the httpmock
func DeActivate() {
	print.Line("RESET")
	urlPrefix = ""
	parent.DeactivateAndReset()
}

// Register registers a httpmock for the given request (the response file is based on the request)
func Register(method string, request string) {
	RegisterWithCode(method, request, 200)
}

// RegisterWithCode is the same as Register but it allows specifying a code
func RegisterWithCode(method string, request string, code int) {
	responseFile := strings.Replace(request, urlPrefix, "", 1)
	RegisterWithResponse(method, request, code, responseFile)
}

// RegisterWithResponse is the same as RegisterWithCode but it allows specifying a response file
func RegisterWithResponse(method string, request string, code int, responseFile string) {
	responsePath := getResponsePath()
	responseFile = getResponseFile(method, code, responseFile, responsePath)
	request = urlPrefix + "/" + strings.TrimPrefix(request, "/")
	parent.RegisterResponder(method, request,
		parent.NewStringResponder(code, string(fileutils.ReadFileUnsafe(responseFile))))
}

// RegisterWithResponder register a httpmock with a custom responder
func RegisterWithResponder(method string, request string, cb func(req *http.Request) (int, string)) {
	request = urlPrefix + "/" + strings.TrimPrefix(request, "/")
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
	responseFile = filepath.Join(responsePath, responseFile) + ".json"

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
