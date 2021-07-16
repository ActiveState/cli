package buildlogstream

import (
	"net/http"

	"github.com/gorilla/websocket"
	"golang.org/x/net/context"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api"
)

func Connect(ctx context.Context) (*websocket.Conn, error) {
	url := api.GetServiceURL(api.BuildLogStreamer)
	header := make(http.Header)
	header.Add("Origin", "https://"+url.Host)

	logging.Debug("Creating websocket for %s (origin: %s)", url.String(), header.Get("Origin"))
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, url.String(), header)
	if err != nil {
		return nil, errs.Wrap(err, "Could not create websocket dialer")
	}
	return conn, nil
}
