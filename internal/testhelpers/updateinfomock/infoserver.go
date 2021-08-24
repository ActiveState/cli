package updateinfomock

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/legacyupd"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/stretchr/testify/suite"
)

// TestPort is the port
var TestPort = "24217"

// MockUpdateInfoRequest represents a request that has been sent to a MockInfoServer instance
type MockUpdateInfoRequest struct {
	suite    suite.Suite
	req      *http.Request
	isLegacy bool
	setTag   string
}

// ExpectQueryParam expects that a query parameter with key key has been set to the expectedValue
func (mur *MockUpdateInfoRequest) ExpectQueryParam(key, expectedValue string) {
	mur.suite.Assert().Equal(expectedValue, mur.req.URL.Query().Get(key), "Unexpected value for query parameter '%s'", key)
}

// ExpectLegacyQuery expects that the query targeted the legacy update endpoint
func (mur *MockUpdateInfoRequest) ExpectLegacyQuery(legacy bool) {
	mur.suite.Assert().Equal(legacy, mur.isLegacy)
}

// ExpectTagResponse expects that the server responded with a tag field in the response
func (mur *MockUpdateInfoRequest) ExpectTagResponse(tag string) {
	mur.suite.Assert().Equal(tag, mur.setTag)
}

// MockUpdateInfoServer serves update information files found relative to a testUpdateDir directory path
type MockUpdateInfoServer struct {
	suite                suite.Suite
	testUpdateDir        string
	close                func()
	updateModifier       func(*updater.AvailableUpdate, string, string)
	legacyUpdateModifier func(*legacyupd.Info, string, string)
	requests             []*MockUpdateInfoRequest
}

// New constructs a new MockUpdateInfoServer serving update files relative to the testUpdateDir directory
func New(suite suite.Suite, testUpdateDir string) *MockUpdateInfoServer {
	s := &MockUpdateInfoServer{
		suite:         suite,
		testUpdateDir: testUpdateDir,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/info/legacy", s.handleLegacyInfo)
	mux.HandleFunc("/info", s.handleInfo)
	mux.Handle("/", http.FileServer(http.Dir(testUpdateDir)))
	logger := log.New(os.Stdout, "http: ", log.LstdFlags)
	server := &http.Server{Addr: "localhost:" + TestPort, Handler: mux, ErrorLog: logger}
	go func() {
		err := server.ListenAndServe()
		if err == http.ErrServerClosed {
			return
		}
		suite.Require().NoError(err)
	}()

	s.close = func() {
		err := server.Shutdown(context.Background())
		suite.Require().NoError(err)
	}
	return s
}

// Close cleans up all resources and stops the server
func (mus *MockUpdateInfoServer) Close() {
	mus.close()
}

// SetLegacyUpdateModifier sets a function modifying the returned update information when a legacy info was requested
func (mus *MockUpdateInfoServer) SetLegacyUpdateModifier(mod func(*legacyupd.Info, string, string)) {
	mus.legacyUpdateModifier = mod
}

// SetLegacyUpdateModifier sets a function modifying the returned update information
func (mus *MockUpdateInfoServer) SetUpdateModifier(mod func(*updater.AvailableUpdate, string, string)) {
	mus.updateModifier = mod
}

// ExpectNRequests ensures that the server handled exactly N requests so far
func (mus *MockUpdateInfoServer) ExpectNRequests(n int) {
	mus.suite.Require().Len(mus.requests, n)
}

// NthRequest returns information about the n-th request for further inspection
func (mus *MockUpdateInfoServer) NthRequest(n int) *MockUpdateInfoRequest {
	if n >= len(mus.requests) {
		return nil
	}

	return mus.requests[n]
}

// MockedUpdateServerEnvVars returns environment variable settings that can be used in integration tests to target the mocked info server
func MockedUpdateServerEnvVars() []string {
	return []string{
		fmt.Sprintf("_TEST_UPDATE_URL=http://localhost:%s", TestPort),
		fmt.Sprintf("_TEST_UPDATE_INFO_URL=http://localhost:%s", TestPort),
	}
}
func (mus *MockUpdateInfoServer) handleInfo(rw http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	platform := q.Get("platform")
	arch := "amd64"
	source := q.Get("source")
	version := q.Get("target-version")
	channel := q.Get("channel")
	tag := q.Get("tag")

	fp := filepath.Join(mus.testUpdateDir, channel)
	if version != "" {
		fp = filepath.Join(fp, version)
	}
	fp = filepath.Join(fp, fmt.Sprintf("%s-%s", platform, arch), "info.json")

	b, err := os.ReadFile(fp)
	mus.suite.Require().NoError(err, "failed finding version info file")

	var up *updater.AvailableUpdate
	err = json.Unmarshal(b, &up)
	mus.suite.Require().NoError(err)

	if mus.updateModifier != nil {
		mus.updateModifier(up, source, tag)
	}

	b, err = json.MarshalIndent(up, "", "")
	mus.suite.Require().NoError(err, "failed marshaling the response")

	fmt.Fprintf(rw, "%s", b)

	mus.requests = append(mus.requests, &MockUpdateInfoRequest{
		suite:    mus.suite,
		req:      r,
		isLegacy: false,
		setTag:   up.Tag,
	})
}

func (mus *MockUpdateInfoServer) handleLegacyInfo(rw http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	platform := q.Get("platform")
	arch := "amd64"
	source := q.Get("source")
	version := q.Get("target-version")
	channel := q.Get("channel")
	tag := q.Get("tag")

	fp := filepath.Join(mus.testUpdateDir, channel)
	if version != "" {
		fp = filepath.Join(fp, version)
	}
	fp = filepath.Join(fp, fmt.Sprintf("%s-%s.json", platform, arch))

	b, err := os.ReadFile(fp)
	mus.suite.Require().NoError(err, "failed finding version info file")

	var up *legacyupd.Info
	err = json.Unmarshal(b, &up)
	mus.suite.Require().NoError(err)

	if mus.legacyUpdateModifier != nil {
		mus.legacyUpdateModifier(up, source, tag)
	}

	b, err = json.MarshalIndent(up, "", "")
	mus.suite.Require().NoError(err, "failed marshaling the response")

	fmt.Fprintf(rw, "%s", b)

	mus.requests = append(mus.requests, &MockUpdateInfoRequest{
		suite:    mus.suite,
		req:      r,
		isLegacy: true,
		setTag:   up.Tag,
	})
}
