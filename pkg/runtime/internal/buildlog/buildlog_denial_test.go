package buildlog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/pkg/buildplan"
	"github.com/ActiveState/cli/pkg/platform/api/buildlogstream"
	"github.com/go-openapi/strfmt"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWait_SoftCloseIsDenial drives Wait against a server that accepts the
// Upgrade then soft-closes with no frames (one of the two deny shapes), and
// asserts the run degrades to a denial instead of surfacing a build failure.
func TestWait_SoftCloseIsDenial(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("mock build-log-streamer failed to upgrade: %v", err)
			return
		}
		// Drain the client's recipe request, then close normally with no build
		// frames -- the server-side soft-close denial shape.
		_, _, _ = conn.ReadMessage()
		_ = conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			time.Now().Add(5*time.Second))
		_ = conn.Close()
	}))
	t.Cleanup(srv.Close)

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	t.Setenv(constants.APIServiceOverrideEnvVarName+"BUILDLOG_STREAMER", wsURL)

	blog := New(strfmt.UUID("00000000-0000-0000-0000-000000000000"), buildplan.ArtifactIDMap{}, "")

	err := blog.Wait(context.Background())
	require.Error(t, err)
	assert.Truef(t, buildlogstream.IsStreamDenied(err), "a soft-close with no frames must be recognized as a denial, got: %v", err)
}
