package api

import (
	"bytes"
	"log"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/alecthomas/template"

	"github.com/ActiveState/cli/pkg/sysinfo"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/retryhttp"
	"github.com/ActiveState/cli/internal/singleton/uniqid"
	"github.com/ActiveState/cli/pkg/platform"
)

// NewHTTPClient creates a new HTTP client that will retry requests and
// add additional request information to the request headers
func NewHTTPClient() *http.Client {
	if condition.InUnitTest() {
		return http.DefaultClient
	}

	return &http.Client{
		Timeout:   5 * time.Second,
		Transport: NewRoundTripper(retryhttp.DefaultClient.StandardClient().Transport),
	}
}

// RoundTripper is an implementation of http.RoundTripper that adds additional request information
type RoundTripper struct {
	transport http.RoundTripper
}

// RoundTrip executes a single HTTP transaction, returning a Response for the provided Request.
func (r *RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("User-Agent", r.UserAgent())
	req.Header.Add("X-Requestor", uniqid.Text())

	resp, err := r.transport.RoundTrip(req)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusForbidden && strings.EqualFold(resp.Header.Get("server"), "cloudfront") {
			return nil, platform.NewCountryBlockedError()
		}

		return resp, err
	}

	return resp, err
}

// UserAgent returns the user agent used by the State Tool
func (r *RoundTripper) UserAgent() string {
	var osVersionStr string
	osVersion, err := sysinfo.OSVersion()
	if err != nil {
		logging.Error("Could not detect OS version: %v", err)
	} else {
		osVersionStr = osVersion.Version
	}

	agentTemplate, err := template.New("").Parse(constants.UserAgentTemplate)
	if err != nil {
		log.Panicf("Parsing user agent template failed: %v", err)
	}

	var userAgent bytes.Buffer
	agentTemplate.Execute(&userAgent, struct {
		UserAgent    string
		OS           string
		OSVersion    string
		Architecture string
	}{
		UserAgent:    constants.UserAgent,
		OS:           sysinfo.OS().String(),
		OSVersion:    osVersionStr,
		Architecture: sysinfo.Architecture().String(),
	})

	return userAgent.String()
}

// NewRoundTripper creates a new instance of RoundTripper
func NewRoundTripper(transport http.RoundTripper) http.RoundTripper {
	return &RoundTripper{transport}
}

// ErrorCode tries to retrieve the code associated with an API error
func ErrorCode(err interface{}) int {
	codeVal := reflect.Indirect(reflect.ValueOf(err)).FieldByName("Code")
	if codeVal.IsValid() {
		return int(codeVal.Int())
	}
	return ErrorCodeFromPayload(err)
}

// ErrorCodeFromPayload tries to retrieve the code associated with an API error from a
// Message object referenced as a Payload.
func ErrorCodeFromPayload(err interface{}) int {
	errVal := reflect.ValueOf(err)
	payloadVal := reflect.Indirect(errVal).FieldByName("Payload")
	if !payloadVal.IsValid() {
		return -1
	}

	codePtr := reflect.Indirect(payloadVal).FieldByName("Code")
	if !codePtr.IsValid() {
		return -1
	}

	codeVal := reflect.Indirect(codePtr)
	if !codeVal.IsValid() {
		return -1
	}
	return int(codeVal.Int())
}

// ErrorMessageFromPayload tries to retrieve the code associated with an API error from a
// Message object referenced as a Payload.
func ErrorMessageFromPayload(err error) string {
	errVal := reflect.ValueOf(err)
	payloadVal := reflect.Indirect(errVal).FieldByName("Payload")
	if !payloadVal.IsValid() {
		return err.Error()
	}

	codePtr := reflect.Indirect(payloadVal).FieldByName("Message")
	if !codePtr.IsValid() {
		return err.Error()
	}

	codeVal := reflect.Indirect(codePtr)
	if !codeVal.IsValid() {
		return err.Error()
	}
	return codeVal.String()
}
