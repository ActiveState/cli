package uniqid

import (
	"sync"

	uid "github.com/ActiveState/cli/internal/uniqid"
)

var (
	mu  sync.Mutex
	def *uid.UniqID
)

// Text returns the stored or new unique id as a string.
func Text() string {
	mu.Lock()
	defer mu.Unlock()

	if def == nil {
		id, err := uid.New(uid.InHome)
		if err != nil {
			// log error
			return ""
		}

		def = id
	}

	return def.String()
}
