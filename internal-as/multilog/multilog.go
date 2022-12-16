package multilog

import (
	"github.com/ActiveState/cli/internal-as/logging"
	"github.com/ActiveState/cli/internal-as/rollbar"
)

type LogFunc func(string, ...interface{})

func Log(fns ...LogFunc) LogFunc {
	return func(format string, args ...interface{}) {
		for _, fn := range fns {
			fn(format, args...)
		}
	}
}

func Error(format string, args ...interface{}) {
	logging.Error(format, args...)
	rollbar.Error(format, args...)
}

func Critical(format string, args ...interface{}) {
	logging.Critical(format, args...)
	rollbar.Critical(format, args...)
}
