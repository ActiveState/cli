package raw

import (
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/go-openapi/strfmt"
)

func (r *Raw) hydrate() error {
	// Locate the AtTime key and define it at the top level for easier access, and to raise errors at hydrate (ie. unmarshal) time
	for _, assignment := range r.Assignments {
		key := assignment.Key
		value := assignment.Value
		if key != AtTimeKey {
			continue
		}
		if value.Str == nil {
			return nil
		}
		atTime, err := strfmt.ParseDateTime(strings.Trim(*value.Str, `"`))
		if err != nil {
			return errs.Wrap(err, "Invalid timestamp: %s", *value.Str)
		}
		r.AtTime = ptr.To(time.Time(atTime))
		return nil
	}

	return nil
}
