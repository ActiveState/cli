package runtime

import (
	"github.com/ActiveState/cli/internal/locale"
)

type ErrUpdate struct {
	*locale.LocalizedError
}

type Configurable interface {
	GetString(key string) string
	GetBool(key string) bool
}
