// Package instanceid exposes some of the instanceid internal package.
package instanceid

import "github.com/ActiveState/cli/internal/instanceid"

func ID() string {
	return instanceid.ID()
}
