package updater

import (
	"context"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
)

// CheckService checks for updates every 30 minutes in the background, and the most recent update response can be requested
type CheckService struct {
	requests chan (chan<- updateResponse)
	ctx      context.Context
}

type updateResponse struct {
	up  *AvailableUpdate
	err error
}

// NewCheckService returns a new CheckService continuously polling for update information
func NewCheckService(cfg Configurable, ctx context.Context) *CheckService {
	requests := make(chan (chan<- updateResponse))
	go func() {
		resp := refreshUpdate(cfg)
		for {
			select {
			case <-time.After(time.Minute * 30):
				resp = refreshUpdate(cfg)
			case req := <-requests:
				req <- resp
			case <-ctx.Done():
				return
			}
		}
	}()

	return &CheckService{requests, ctx}
}

// LatestUpdate returns the latest update response that has been queried in the background
func (s *CheckService) LatestUpdate() (*AvailableUpdate, error) {
	resp := make(chan updateResponse)
	defer close(resp)
	select {
	case s.requests <- resp:
	case <-s.ctx.Done():
		return nil, errs.New("Update checking service has been terminated already.")
	}
	r := <-resp
	return r.up, r.err
}

func refreshUpdate(cfg Configurable) updateResponse {
	up, err := NewDefaultChecker(cfg).Check()
	if err != nil {
		logging.Error("Failed to check for latest update in state-svc: %s", errs.JoinMessage(err))
	}
	logging.Debug("Available update result is %v", up)
	return updateResponse{up, err}
}
