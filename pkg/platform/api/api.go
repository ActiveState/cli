package api

import (
	"bytes"
	"flag"
	"log"
	"net/http"
	"reflect"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api/client"
	"github.com/ActiveState/sysinfo"
	"github.com/alecthomas/template"
	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
)

// persist contains the active API Client connection
var persist *client.APIClient

var (
	// FailUnknown is the failure type used for API requests with an unexpected error
	FailUnknown = failures.Type("api.fail.unknown")

	// FailAuth is the failure type used for failed authentication API requests
	FailAuth = failures.Type("api.fail.auth", failures.FailUser)

	// FailNotFound indicates a failure to find a user's resource.
	FailNotFound = failures.Type("api.fail.not_found", failures.FailUser)

	// FailOrganizationNotFound is used when a project could not be found
	FailOrganizationNotFound = failures.Type("api.fail.organization.not_found", FailNotFound)

	// FailProjectNotFound is used when a project could not be found
	FailProjectNotFound = failures.Type("api.fail.project.not_found", FailNotFound)
)

// New will create a new API client using default settings (for an authenticated version use the NewWithAuth version)
func New() *client.APIClient {
	return Init(GetSettings(ServicePlatform), nil)
}

// NewWithAuth creates a new API client using default settings and the provided authentication info
func NewWithAuth(auth *runtime.ClientAuthInfoWriter) *client.APIClient {
	return Init(GetSettings(ServicePlatform), auth)
}

// Init initializes a new api client
func Init(apiSetting Settings, auth *runtime.ClientAuthInfoWriter) *client.APIClient {
	transportRuntime := httptransport.New(apiSetting.Host, apiSetting.BasePath, []string{apiSetting.Schema})
	transportRuntime.Transport = NewUserAgentTripper()

	if flag.Lookup("test.v") != nil {
		transportRuntime.SetDebug(true)
	}

	if auth != nil {
		transportRuntime.DefaultAuthentication = *auth
	}
	return client.New(transportRuntime, strfmt.Default)
}

// Get returns a cached version of the default api client
func Get() *client.APIClient {
	if persist == nil {
		persist = New()
	}
	return persist
}

// UserAgentTripper is an implementation of http.RoundTripper that adds our useragent
type UserAgentTripper struct{}

// RoundTrip executes a single HTTP transaction, returning a Response for the provided Request.
func (r *UserAgentTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", r.UserAgent())
	return http.DefaultTransport.RoundTrip(req)
}

// UserAgent returns the user agent used by the state tool
func (r *UserAgentTripper) UserAgent() string {
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

// NewUserAgentTripper creates a new instance of UserAgentTripper
func NewUserAgentTripper() http.RoundTripper {
	return &UserAgentTripper{}
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
