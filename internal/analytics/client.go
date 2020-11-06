package analytics

import (
	"net/http"
	"sync"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/loghttp"
	ga "github.com/ActiveState/go-ogle-analytics"
)

func (d *deferSend) get() bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.b
}

type client struct {
	*ga.Client
	deferSend deferSend
}

func newClient(logFn loghttp.LogFunc, uniqID string) (*client, error) {
	gac, err := ga.NewClient(constants.AnalyticsTrackingID)
	if err != nil {
		return nil, errs.Wrap(err, "Cannot initialize GA analytics")
	}

	gac.ClientID(uniqID)
	gac.HttpClient = &http.Client{
		Transport: loghttp.NewTransport(logFn),
		Timeout:   time.Second * 30,
	}

	c := client{Client: gac}

	return &c, nil
}

func (c *client) sendPageview(p *ga.Pageview) error {
	return c.Client.Send(p)
}

func (c *client) sendEvent(e *ga.Event) error {
	return c.Client.Send(e)
}

type deferSend struct {
	b  bool
	mu sync.Mutex
}

func (d *deferSend) set(b bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.b = b
}
