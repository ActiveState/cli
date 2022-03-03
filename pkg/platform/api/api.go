package api

import (
	"bytes"
	"context"
	"log"
	"net"
	"net/http"
	"reflect"

	"github.com/alecthomas/template"

	"github.com/ActiveState/cli/pkg/sysinfo"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/machineid"
)

// RoundTripper is an implementation of http.RoundTripper that adds additional request information
type RoundTripper struct {
	transport *http.Transport
}

// RoundTrip executes a single HTTP transaction, returning a Response for the provided Request.
func (r *RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", r.UserAgent())
	req.Header.Set("X-Requestor", machineid.UniqID())
	return r.transport.RoundTrip(req)
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
	// Force use of IPv4. This is needed particularly for device authorization, which compares the IP
	// address of the State Tool request and the IP address of the web client request. If there's a
	// mismatch, authorization fails. When this happens, it's often because the State Tool connects to
	// the Platform via IPv6, but the browser does via IPv4 (browsers apparently prefer IPv4 for now).
	var dialer net.Dialer
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return dialer.DialContext(ctx, "tcp4", addr)
	}
	return &RoundTripper{transport} // or http.DefaultTransport.(*http.Transport) for allowing IPv6
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
