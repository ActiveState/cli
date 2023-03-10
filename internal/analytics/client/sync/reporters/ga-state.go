package reporters

import (
	"strconv"

	anaConsts "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/analytics/dimensions"
	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/ptr"
	ga "github.com/ActiveState/go-ogle-analytics"
)

type GaCLIReporter struct {
	ga   *ga.Client
	omit map[string]struct{}
}

func NewGaCLIReporter(clientID string) (*GaCLIReporter, error) {
	r := &GaCLIReporter{
		omit: make(map[string]struct{}),
	}

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

func (r *GaCLIReporter) AddOmitCategory(category string) {
	r.omit[category] = struct{}{}
}

func (r *GaCLIReporter) Event(category, action, label string, d *dimensions.Values) error {
	if _, ok := r.omit[category]; ok {
		logging.Debug("Not sending event with category: %s to Google Analytics", category)
		return nil
	}

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
		if condition.IsNetworkingError(err) {
			logging.Debug("Cannot send Google Analytics event as the hostname appears to be blocked. Error received: %s", err.Error())
			return nil
		}
		return errs.Wrap(err, "Could not send GA Event")
	}

	return nil
}

func legacyDimensionMap(d *dimensions.Values) map[string]string {
	return map[string]string{
		// Commented out idx 1 so it's clear why we start with 2. We used to log the hostname while dogfooding internally.
		// "1": "hostname (deprecated)"
		"2": ptr.Deref(d.Version),
		"3": ptr.Deref(d.BranchName),
		"4": ptr.Deref(d.UserID),
		"5": ptr.Deref(d.OutputType),
		"6": ptr.Deref(d.OSName),
		"7": ptr.Deref(d.OSVersion),
		"8": ptr.Deref(d.InstallSource),
		// "9":  "machineID (deprecated in favor of uniqID)"
		"10": ptr.Deref(d.ProjectNameSpace),
		"11": ptr.Deref(d.SessionToken),
		"12": ptr.Deref(d.UniqID),
		"13": ptr.Deref(d.UpdateTag),
		"14": ptr.Deref(d.ProjectID),
		"16": ptr.Deref(d.Trigger),
		"17": ptr.Deref(d.InstanceID),
		"18": ptr.Deref(d.Headless),
		"19": ptr.Deref(d.CommitID),
		"20": ptr.Deref(d.Command),
		"21": strconv.Itoa(ptr.DerefInt(d.Sequence)),
	}
}
