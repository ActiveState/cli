package messages

import (
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/pkg/sysinfo"
	"github.com/blang/semver"
	"github.com/thoas/go-funk"
)

type ConditionParams struct {
	UserID       string
	UserName     string
	UserEmail    string
	OS           string
	OSVersion    Version
	StateChannel string
	StateVersion Version
	Command      string
	Flags        []string
}

type Version struct {
	Raw   string
	Major int
	Minor int
	Patch int
	Build string
}

func NewVersionFromSemver(v semver.Version) Version {
	return Version{
		Raw:   v.String(),
		Major: int(v.Major),
		Minor: int(v.Minor),
		Patch: int(v.Patch),
	}
}

func NewVersionFromSysinfo(osVersion *sysinfo.OSVersionInfo) Version {
	return Version{
		Raw:   osVersion.Version,
		Major: osVersion.Major,
		Minor: osVersion.Minor,
		Patch: osVersion.Micro,
	}
}

func conditionFuncMap() template.FuncMap {
	return map[string]interface{}{
		"contains":  funk.Contains,
		"hasSuffix": strings.HasSuffix,
		"hasPrefix": strings.HasPrefix,
		"dateBefore": func(date string) bool {
			parsedDate, err := time.Parse(time.RFC3339, date)
			if err != nil {
				multilog.Error("Messages: Could not parse date: %s", err)
				return false
			}
			return time.Now().Before(parsedDate)
		},
		"dateAfter": func(date string) bool {
			parsedDate, err := time.Parse(time.RFC3339, date)
			if err != nil {
				multilog.Error("Messages: Could not parse date: %s", err)
				return false
			}
			return time.Now().After(parsedDate)
		},
		"regexMatch": func(str, pattern string) bool {
			rx, err := regexp.Compile(pattern)
			if err != nil {
				multilog.Error("Messages: Could not compile regex pattern: %s", err)
				return false
			}
			return rx.MatchString(str)
		},
	}
}
