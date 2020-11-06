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

type sender interface {
	send(c *ga.Client) error
	// json marshal/unmarshal
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

func (c *client) send(s sender) error {
	if c.deferSend.get() {
		// append event data to file
		// return err/nil
	}

	// otherwise, get slice of senders from file
	// range slice of senders
	//  // s.send(c.Client, c.deferSend.get())
	//  // halt loop on first error and store

	// s.send current
	// if err, return this err, if nil, return loop err
	return s.send(c.Client)
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

func (d *deferSend) get() bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.b
}
