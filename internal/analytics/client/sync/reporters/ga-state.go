package reporters

import (
	anaConsts "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/analytics/dimensions"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/rtutils/p"
	ga "github.com/ActiveState/go-ogle-analytics"
)

type GaCLIReporter struct {
	ga *ga.Client
}

func NewGaCLIReporter(clientID string) (*GaCLIReporter, error) {
	r := &GaCLIReporter{}

	trackingID := constants.AnalyticsTrackingID

	client, err := ga.NewClient(trackingID)
	if err != nil {
		return nil, errs.Wrap(err, "Cannot initialize google analytics cli client")
	}

	client.ClientID(clientID)
	r.ga = client

	return r, nil
}

func (r *GaCLIReporter) ID() string {
	return "GaCLIReporter"
}

func (r *GaCLIReporter) Event(category, action, label string, d *dimensions.Values) error {
	r.ga.CustomDimensionMap(legacyDimensionMap(d))

	if category == anaConsts.CatRunCmd {
		r.ga.Send(ga.NewPageview())
	}
	event := ga.NewEvent(category, action)
	if label != "" {
		event.Label(label)
	}
	err := r.ga.Send(event)
	if err != nil {
		return errs.Wrap(err, "Could not send GA Event")
	}

	return nil
}

func legacyDimensionMap(d *dimensions.Values) map[string]string {
	return map[string]string{
		// Commented out idx 1 so it's clear why we start with 2. We used to log the hostname while dogfooding internally.
		// "1": "hostname (deprected)"
		"2":  p.PStr(d.Version),
		"3":  p.PStr(d.BranchName),
		"4":  p.PStr(d.UserID),
		"5":  p.PStr(d.OutputType),
		"6":  p.PStr(d.OSName),
		"7":  p.PStr(d.OSVersion),
		"8":  p.PStr(d.InstallSource),
		"9":  p.PStr(d.MachineID),
		"10": p.PStr(d.ProjectNameSpace),
		"11": p.PStr(d.SessionToken),
		"12": p.PStr(d.UniqID),
		"13": p.PStr(d.UpdateTag),
		"14": p.PStr(d.ProjectID),
		"16": p.PStr(d.Trigger),
	}
}
