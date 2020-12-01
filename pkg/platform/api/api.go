package api

import (
	"bytes"
	"log"
	"net/http"
	"reflect"

	"github.com/alecthomas/template"

	"github.com/ActiveState/sysinfo"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/machineid"
)

var (
	// FailUnknown is the failure type used for API requests with an unexpected error
	FailUnknown = failures.Type("api.fail.unknown")

	// FailAuth is the failure type used for failed authentication API requests
	FailAuth = failures.Type("api.fail.auth", failures.FailUser)

	// FailForbidden is the failure type used when access to a requested resource is forbidden
	FailForbidden = failures.Type("api.fail.forbidden", failures.FailUser)

	// FailNotFound indicates a failure to find a user's resource.
	FailNotFound = failures.Type("api.fail.not_found", failures.FailUser)

	// FailOrganizationNotFound is used when an organization could not be found
	FailOrganizationNotFound = failures.Type("api.fail.organization.not_found", FailNotFound)

	// FailProjectNotFound is used when a project could not be found
	FailProjectNotFound = failures.Type("api.fail.project.not_found", FailNotFound)
)

// RoundTripper is an implementation of http.RoundTripper that adds additional request information
type RoundTripper struct{}

// RoundTrip executes a single HTTP transaction, returning a Response for the provided Request.
func (r *RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", r.UserAgent())
	req.Header.Set("X-Requestor", machineid.UniqID())
	return http.DefaultTransport.RoundTrip(req)
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
func NewRoundTripper() http.RoundTripper {
	return &RoundTripper{}
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
