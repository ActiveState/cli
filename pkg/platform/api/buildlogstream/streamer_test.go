package buildlogstream

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// startMockBLS stands up a real WebSocket server that records the
// Sec-WebSocket-Protocol values offered on the Upgrade, and redirects
// Connect's resolved service URL at it via the per-service override env var
// honored by api.GetServiceURL. Returns a pointer that holds the offered
// subprotocols after Connect runs.
func startMockBLS(t *testing.T) *[]string {
	t.Helper()
	offered := &[]string{}

	upgrader := websocket.Upgrader{
		CheckOrigin: func(*http.Request) bool { return true },
		// Echo back only the "real" subprotocol; the bearer.<jwt> entry must
		// not be selected (mirrors the build-log-streamer's allow-list).
		Subprotocols: []string{wsSubprotocol},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*offered = r.Header.Values("Sec-WebSocket-Protocol")
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		_ = conn.Close()
	}))
	t.Cleanup(srv.Close)

	// http://127.0.0.1:port -> ws://127.0.0.1:port
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	t.Setenv(constants.APIServiceOverrideEnvVarName+"BUILDLOG_STREAMER", wsURL)

	return offered
}

func TestConnect_ForwardsJWTViaSubprotocol(t *testing.T) {
	offered := startMockBLS(t)

	conn, err := Connect(context.Background(), "header.payload.signature")
	require.NoError(t, err)
	_ = conn.Close()

	joined := strings.Join(*offered, ",")
	assert.Contains(t, joined, "bearer.header.payload.signature",
		"client must offer the JWT as a bearer.<jwt> subprotocol")
	assert.Contains(t, joined, wsSubprotocol,
		"client must still offer the real subprotocol the server echoes back")
}

func TestConnect_AnonymousOffersNoBearer(t *testing.T) {
	offered := startMockBLS(t)

	conn, err := Connect(context.Background(), "")
	require.NoError(t, err)
	_ = conn.Close()

	joined := strings.Join(*offered, ",")
	assert.NotContains(t, joined, "bearer.",
		"anonymous Connect must not offer a bearer subprotocol")
	assert.Contains(t, joined, wsSubprotocol,
		"anonymous Connect must still offer the real subprotocol")
}
