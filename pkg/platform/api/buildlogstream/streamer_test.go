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

// upgradeRequest captures the headers the build-log-streamer server saw on the
// WS Upgrade.
type upgradeRequest struct {
	protocols []string
	userAgent string
}

// startMockBLS stands up a real WebSocket server that records the Upgrade
// request headers, and redirects Connect's resolved service URL at it via the
// per-service override env var honored by api.GetServiceURL. Returns a pointer
// populated after Connect runs.
func startMockBLS(t *testing.T) *upgradeRequest {
	t.Helper()
	got := &upgradeRequest{}

	upgrader := websocket.Upgrader{
		CheckOrigin: func(*http.Request) bool { return true },
		// Echo back only the "real" subprotocol; the bearer.<jwt> entry must
		// not be selected (mirrors the build-log-streamer's allow-list).
		Subprotocols: []string{wsSubprotocol},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.protocols = r.Header.Values("Sec-WebSocket-Protocol")
		got.userAgent = r.Header.Get("User-Agent")
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

	return got
}

func TestConnect_ForwardsJWTViaSubprotocol(t *testing.T) {
	got := startMockBLS(t)

	conn, err := Connect(context.Background(), "header.payload.signature")
	require.NoError(t, err)
	_ = conn.Close()

	joined := strings.Join(got.protocols, ",")
	assert.Contains(t, joined, "bearer.header.payload.signature",
		"client must offer the JWT as a bearer.<jwt> subprotocol")
	assert.Contains(t, joined, wsSubprotocol,
		"client must still offer the real subprotocol the server echoes back")
	assert.Contains(t, got.userAgent, "state/",
		"client must send the versioned State Tool User-Agent so the server can monitor versions")
}

func TestConnect_AnonymousOffersNoBearer(t *testing.T) {
	got := startMockBLS(t)

	conn, err := Connect(context.Background(), "")
	require.NoError(t, err)
	_ = conn.Close()

	joined := strings.Join(got.protocols, ",")
	assert.NotContains(t, joined, "bearer.",
		"anonymous Connect must not offer a bearer subprotocol")
	assert.Contains(t, joined, wsSubprotocol,
		"anonymous Connect must still offer the real subprotocol")
	assert.Contains(t, got.userAgent, "state/",
		"client must send the versioned State Tool User-Agent even when anonymous")
}
