package buildlogstream

import (
	"net/http"

	"github.com/gorilla/websocket"
	"golang.org/x/net/context"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api"
)

// wsSubprotocol is the "real" subprotocol the build-log-streamer echoes back.
// The server's upgrader allow-list contains only this value, so the
// bearer.<jwt> entry we also offer never appears in the upgrade response,
// keeping the token out of proxy/browser response logs.
const wsSubprotocol = "build-log-streamer.activestate.com.v1"

// Connect opens the build-log-streamer WebSocket. When jwt is non-empty it is
// offered via Sec-WebSocket-Protocol as `bearer.<jwt>` (alongside
// wsSubprotocol, which the server echoes back) so the server can authorize the
// stream. The browser WebSocket API can't set custom request headers, so the
// dashboard carries the JWT the same way; using the subprotocol here keeps
// both clients symmetric. See ENG-1372 / ENG-1374.
func Connect(ctx context.Context, jwt string) (*websocket.Conn, error) {
	url := api.GetServiceURL(api.BuildLogStreamer)
	header := make(http.Header)
	header.Add("Origin", "https://"+url.Host)
	// Send the versioned State Tool User-Agent so the server can monitor which
	// State Tool versions are connecting (e.g. to size the unauthenticated tail
	// before tightening the gate). See ENG-1372.
	header.Set("User-Agent", api.UserAgent())

	dialer := *websocket.DefaultDialer // copy so we don't mutate the package global
	dialer.Subprotocols = []string{wsSubprotocol}
	if jwt != "" {
		dialer.Subprotocols = []string{"bearer." + jwt, wsSubprotocol}
	}

	logging.Debug("Creating websocket for %s (origin: %s)", url.String(), header.Get("Origin"))
	conn, _, err := dialer.DialContext(ctx, url.String(), header)
	if err != nil {
		return nil, errs.Wrap(err, "Could not create websocket dialer")
	}
	return conn, nil
}
