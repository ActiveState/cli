package rtstat

import (
	"github.com/ActiveState/cli/internal/analytics"
)

const (
	runtime     = "runtime"
	actBuild    = "build"
	actCache    = "cache"
	actDownload = "download"
	actStart    = "start"
	actSuccess  = "success"
	actFailure  = "failure"
)

// RtStat contains info relevant to an analytics event send.
type RtStat struct {
	action string
	label  string
}

// Send will fire the correct anaylitcs event call.
func (stat RtStat) Send() {
	if stat.label != "" {
		analytics.EventWithLabel(runtime, stat.action, stat.label)
		return
	}
	analytics.Event(runtime, stat.action)
}

// String implements the fmt.Stringer interface.
func (stat RtStat) String() string {
	s := stat.action
	if stat.label != "" {
		s += ":" + stat.action
	}
	return s
}

// RtStat vars provide convenient access to available runtime analytics event
// types.
var (
	Build        = RtStat{action: actBuild}
	Cache        = RtStat{action: actCache}
	Download     = RtStat{action: actDownload}
	Start        = RtStat{action: actStart}
	Success      = RtStat{action: actSuccess}
	FailBuild    = RtStat{action: actFailure, label: actBuild}
	FailDownload = RtStat{action: actFailure, label: actDownload}
)
